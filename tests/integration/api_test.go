package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/suwarnold/DigitalLendingPlatform/internal/app"
	"github.com/suwarnold/DigitalLendingPlatform/internal/borrower"
	"github.com/suwarnold/DigitalLendingPlatform/internal/outbox"
	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/auth"
	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/database"
	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/idempotency"
	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/logger"
)

func TestBorrowerHappyPath(t *testing.T) {
	srv, db := newTestServer(t)
	phone := uniquePhone()

	reg := doJSON(t, srv, http.MethodPost, "/api/v1/borrowers", "", map[string]any{
		"full_name":    "Ada Lovelace",
		"phone_number": phone,
		"email":        "ada@example.com",
		"password":     "supersecret",
	}, nil)
	if reg.StatusCode != http.StatusCreated {
		t.Fatalf("register status = %d body = %s", reg.StatusCode, reg.Body)
	}
	if reg.JSON["kyc_status"] != "PENDING" {
		t.Fatalf("kyc status = %v", reg.JSON["kyc_status"])
	}

	login := doJSON(t, srv, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"phone_number": phone,
		"password":     "supersecret",
	}, nil)
	if login.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d body = %s", login.StatusCode, login.Body)
	}
	token, _ := login.JSON["access_token"].(string)
	if token == "" {
		t.Fatal("missing access token")
	}

	me := doJSON(t, srv, http.MethodGet, "/api/v1/borrowers/me", token, nil, nil)
	if me.StatusCode != http.StatusOK {
		t.Fatalf("me status = %d body = %s", me.StatusCode, me.Body)
	}
	if me.JSON["phone_number"] != "+62"+phone[1:] && me.JSON["phone_number"] != "+628"+phone[2:] {
		if got, _ := me.JSON["phone_number"].(string); got[:3] != "+62" {
			t.Fatalf("phone = %v", me.JSON["phone_number"])
		}
	}

	verify := doJSON(t, srv, http.MethodPost, "/api/v1/borrowers/verify-identity", token, map[string]any{}, nil)
	if verify.StatusCode != http.StatusOK {
		t.Fatalf("verify status = %d body = %s", verify.StatusCode, verify.Body)
	}
	if verify.JSON["kyc_status"] != "IN_REVIEW" {
		t.Fatalf("kyc status after verify = %v", verify.JSON["kyc_status"])
	}

	again := doJSON(t, srv, http.MethodPost, "/api/v1/borrowers/verify-identity", token, map[string]any{}, nil)
	if again.StatusCode != http.StatusOK || again.JSON["kyc_status"] != "IN_REVIEW" {
		t.Fatalf("repeat verify = %d %s", again.StatusCode, again.Body)
	}

	var unpublished int
	if err := db.Get(&unpublished, `SELECT count(*) FROM outbox_events WHERE published_at IS NULL`); err != nil {
		t.Fatalf("count outbox: %v", err)
	}
	if unpublished < 2 {
		t.Fatalf("expected unpublished outbox events, got %d", unpublished)
	}
}

func TestRegisterConflictAndAuthFailures(t *testing.T) {
	srv, _ := newTestServer(t)
	phone := uniquePhone()
	body := map[string]any{
		"full_name":    "Ada Lovelace",
		"phone_number": phone,
		"password":     "supersecret",
	}

	first := doJSON(t, srv, http.MethodPost, "/api/v1/borrowers", "", body, nil)
	if first.StatusCode != http.StatusCreated {
		t.Fatalf("register status = %d body = %s", first.StatusCode, first.Body)
	}
	dup := doJSON(t, srv, http.MethodPost, "/api/v1/borrowers", "", body, nil)
	if dup.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate status = %d body = %s", dup.StatusCode, dup.Body)
	}

	badLogin := doJSON(t, srv, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"phone_number": phone,
		"password":     "wrong-password",
	}, nil)
	if badLogin.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bad login status = %d", badLogin.StatusCode)
	}

	unauth := doJSON(t, srv, http.MethodGet, "/api/v1/borrowers/me", "", nil, nil)
	if unauth.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauth me status = %d", unauth.StatusCode)
	}
}

func TestRegisterIdempotency(t *testing.T) {
	srv, _ := newTestServer(t)
	phone := uniquePhone()
	headers := map[string]string{"Idempotency-Key": uuid.NewString()}
	body := map[string]any{
		"full_name":    "Ada Lovelace",
		"phone_number": phone,
		"password":     "supersecret",
	}

	first := doJSON(t, srv, http.MethodPost, "/api/v1/borrowers", "", body, headers)
	if first.StatusCode != http.StatusCreated {
		t.Fatalf("first register = %d %s", first.StatusCode, first.Body)
	}
	second := doJSON(t, srv, http.MethodPost, "/api/v1/borrowers", "", body, headers)
	if second.StatusCode != http.StatusCreated {
		t.Fatalf("replay register = %d %s", second.StatusCode, second.Body)
	}
	if first.JSON["id"] != second.JSON["id"] {
		t.Fatalf("idempotent replay changed id %v vs %v", first.JSON["id"], second.JSON["id"])
	}

	conflictBody := map[string]any{
		"full_name":    "Someone Else",
		"phone_number": uniquePhone(),
		"password":     "supersecret",
	}
	conflict := doJSON(t, srv, http.MethodPost, "/api/v1/borrowers", "", conflictBody, headers)
	if conflict.StatusCode != http.StatusConflict {
		t.Fatalf("idempotency conflict = %d %s", conflict.StatusCode, conflict.Body)
	}
}

func TestHealthAndReady(t *testing.T) {
	srv, _ := newTestServer(t)

	health := doJSON(t, srv, http.MethodGet, "/healthz", "", nil, nil)
	if health.StatusCode != http.StatusOK || health.JSON["status"] != "ok" {
		t.Fatalf("health = %d %s", health.StatusCode, health.Body)
	}
	ready := doJSON(t, srv, http.MethodGet, "/readyz", "", nil, nil)
	if ready.StatusCode != http.StatusOK || ready.JSON["status"] != "ready" {
		t.Fatalf("ready = %d %s", ready.StatusCode, ready.Body)
	}

	metricsResp, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatalf("metrics: %v", err)
	}
	defer metricsResp.Body.Close()
	if metricsResp.StatusCode != http.StatusOK {
		t.Fatalf("metrics status = %d", metricsResp.StatusCode)
	}

	docsResp := doJSON(t, srv, http.MethodGet, "/docs", "", nil, nil)
	if docsResp.StatusCode != http.StatusOK || !strings.Contains(docsResp.Body, "swagger-ui") {
		t.Fatalf("docs = %d %s", docsResp.StatusCode, docsResp.Body)
	}
	specResp := doJSON(t, srv, http.MethodGet, "/openapi.yaml", "", nil, nil)
	if specResp.StatusCode != http.StatusOK || !strings.Contains(specResp.Body, "Digital Lending Platform API") {
		t.Fatalf("openapi = %d %s", specResp.StatusCode, specResp.Body)
	}
}

func TestOutboxWorkerPublishes(t *testing.T) {
	_, db := newTestServer(t)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx, `
		INSERT INTO outbox_events (aggregate_type, aggregate_id, event_type, payload)
		VALUES ('borrower', $1, 'BorrowerRegistered', '{"borrower_id":"00000000-0000-0000-0000-000000000001"}')
	`, uuid.New()); err != nil {
		t.Fatalf("insert outbox: %v", err)
	}

	w := &outbox.Worker{
		DB:        db,
		Repo:      outbox.NewRepository(db),
		Publisher: outbox.NoopPublisher{Logger: logger.New("error")},
		Logger:    logger.New("error"),
		BatchSize: 10,
	}
	n, err := w.ProcessOnce(ctx)
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if n == 0 {
		t.Fatal("expected at least one published event")
	}

	var remaining int
	if err := db.Get(&remaining, `SELECT count(*) FROM outbox_events WHERE published_at IS NULL`); err != nil {
		t.Fatalf("count: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("unpublished remaining = %d", remaining)
	}
}

type jsonResp struct {
	StatusCode int
	Body       string
	JSON       map[string]any
}

func doJSON(t *testing.T, srv *httptest.Server, method, path, token string, payload any, headers map[string]string) jsonResp {
	t.Helper()

	var reader io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, srv.URL+path, reader)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	out := jsonResp{StatusCode: resp.StatusCode, Body: string(body), JSON: map[string]any{}}
	if len(body) > 0 {
		_ = json.Unmarshal(body, &out.JSON)
	}
	return out
}

func uniquePhone() string {
	return fmt.Sprintf("0812%08d", time.Now().UnixNano()%100000000)
}

func newTestServer(t *testing.T) (*httptest.Server, *sqlx.DB) {
	t.Helper()
	db := openTestDB(t)

	tokens := auth.NewTokens("integration-test-secret", time.Hour)
	svc := &borrower.Service{
		DB:          db,
		Borrowers:   borrower.NewRepository(),
		Outbox:      outbox.NewRepository(db),
		Idempotency: idempotency.NewRepository(),
		Tokens:      tokens,
	}
	handler := app.NewRouter(app.RouterDeps{
		DB:              db,
		Auth:            tokens,
		BorrowerHandler: &borrower.Handler{Service: svc},
	})
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv, db
}

func openTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	db, err := database.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
