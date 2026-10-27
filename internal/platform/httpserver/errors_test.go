package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/appctx"
	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/apperr"
)

func TestMapErrorValidation(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/borrowers", nil)
	req = req.WithContext(appctx.WithRequestID(req.Context(), "req-1"))
	rec := httptest.NewRecorder()

	MapError(rec, req, apperr.Validation("full_name is required"))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}

	var body errorBody
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Code != "VALIDATION_ERROR" || body.Message != "full_name is required" || body.RequestID != "req-1" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestMapErrorConflictKinds(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/borrowers", nil)
	rec := httptest.NewRecorder()
	MapError(rec, req, apperr.ErrIdempotencyConflict)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d", rec.Code)
	}
}
