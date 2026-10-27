package app

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jmoiron/sqlx"

	"github.com/suwarnold/DigitalLendingPlatform/internal/borrower"
	"github.com/suwarnold/DigitalLendingPlatform/internal/docs"
	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/auth"
	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/httpserver"
	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/metrics"
)

type RouterDeps struct {
	DB              *sqlx.DB
	Auth            *auth.Tokens
	BorrowerHandler *borrower.Handler
}

func NewRouter(deps RouterDeps) http.Handler {
	r := chi.NewRouter()
	r.Use(httpserver.RequestID)
	r.Use(httpserver.SecureHeaders)
	r.Use(httpserver.MaxBytes)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))
	r.Use(metrics.Middleware)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		httpserver.WriteJSON(w, r, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/readyz", readyHandler(deps.DB))
	r.Handle("/metrics", metrics.Handler())
	docs.Mount(r)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/borrowers", deps.BorrowerHandler.Register)
		r.Post("/auth/login", deps.BorrowerHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(httpserver.RequireBorrower(deps.Auth))
			r.Get("/borrowers/me", deps.BorrowerHandler.Me)
			r.Post("/borrowers/verify-identity", deps.BorrowerHandler.VerifyIdentity)
		})
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		httpserver.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "route not found")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		httpserver.WriteError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	})

	return r
}

func readyHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			httpserver.WriteError(w, r, http.StatusServiceUnavailable, "NOT_READY", "database unavailable")
			return
		}
		httpserver.WriteJSON(w, r, http.StatusOK, map[string]string{"status": "ready"})
	}
}
