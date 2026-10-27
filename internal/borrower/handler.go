package borrower

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/appctx"
	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/apperr"
	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/httpserver"
)

type Handler struct {
	Service *Service
}

type registerRequest struct {
	FullName    string  `json:"full_name"`
	PhoneNumber string  `json:"phone_number"`
	Email       *string `json:"email"`
	Password    string  `json:"password"`
}

type loginRequest struct {
	PhoneNumber string `json:"phone_number"`
	Password    string `json:"password"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	var req registerRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid JSON body")
		return
	}

	idemKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if len(idemKey) > 255 {
		httpserver.MapError(w, r, apperr.Validation("Idempotency-Key is too long"))
		return
	}

	_, status, body, err := h.Service.Register(r.Context(), RegisterInput{
		FullName:    req.FullName,
		PhoneNumber: req.PhoneNumber,
		Email:       req.Email,
		Password:    req.Password,
	}, idemKey, hashRequest(raw))
	if err != nil {
		httpserver.MapError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if rid := appctx.RequestID(r.Context()); rid != "" {
		w.Header().Set("X-Request-ID", rid)
	}
	w.WriteHeader(status)
	_, _ = w.Write(append(body, '\n'))
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid JSON body")
		return
	}

	tokens, err := h.Service.Login(r.Context(), LoginInput{
		PhoneNumber: req.PhoneNumber,
		Password:    req.Password,
	})
	if err != nil {
		httpserver.MapError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, tokens)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	id, ok := appctx.BorrowerID(r.Context())
	if !ok {
		httpserver.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	b, err := h.Service.GetByID(r.Context(), id)
	if err != nil {
		httpserver.MapError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, b.Public())
}

func (h *Handler) VerifyIdentity(w http.ResponseWriter, r *http.Request) {
	id, ok := appctx.BorrowerID(r.Context())
	if !ok {
		httpserver.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	b, err := h.Service.VerifyIdentity(r.Context(), id)
	if err != nil {
		httpserver.MapError(w, r, err)
		return
	}
	httpserver.WriteJSON(w, r, http.StatusOK, b.Public())
}

func hashRequest(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
