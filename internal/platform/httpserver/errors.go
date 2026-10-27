package httpserver

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/appctx"
	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/apperr"
)

type errorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

func WriteError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	WriteJSON(w, r, status, errorBody{
		Code:      code,
		Message:   message,
		RequestID: appctx.RequestID(r.Context()),
	})
}

func WriteJSON(w http.ResponseWriter, r *http.Request, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	if rid := appctx.RequestID(r.Context()); rid != "" {
		w.Header().Set("X-Request-ID", rid)
	}
	w.WriteHeader(status)
	if body == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.ErrorContext(r.Context(), "write json response", "error", err)
	}
}

func MapError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, apperr.ErrValidation):
		WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, apperr.ErrUnauthorized):
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
	case errors.Is(err, apperr.ErrForbidden):
		WriteError(w, r, http.StatusForbidden, "FORBIDDEN", err.Error())
	case errors.Is(err, apperr.ErrNotFound):
		WriteError(w, r, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, apperr.ErrIdempotencyConflict):
		WriteError(w, r, http.StatusConflict, "IDEMPOTENCY_CONFLICT", "idempotency key reused with a different request")
	case errors.Is(err, apperr.ErrConflict):
		WriteError(w, r, http.StatusConflict, "CONFLICT", err.Error())
	default:
		slog.ErrorContext(r.Context(), "unhandled error", "error", err, "request_id", appctx.RequestID(r.Context()))
		WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
