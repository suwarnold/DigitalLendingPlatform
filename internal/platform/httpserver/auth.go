package httpserver

import (
	"net/http"
	"strings"

	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/appctx"
	"github.com/suwarnold/DigitalLendingPlatform/internal/platform/auth"
)

func RequireBorrower(tokens *auth.Tokens) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			scheme, token, ok := strings.Cut(header, " ")
			if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
				WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
				return
			}

			claims, err := tokens.Parse(strings.TrimSpace(token))
			if err != nil {
				WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired token")
				return
			}

			ctx := appctx.WithBorrowerID(r.Context(), claims.BorrowerID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
