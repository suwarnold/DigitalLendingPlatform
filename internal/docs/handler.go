package docs

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/suwarnold/DigitalLendingPlatform/api"
)

func Mount(r chi.Router) {
	r.Get("/openapi.yaml", ServeSpec)
	r.Get("/docs", ServeUI)
	r.Get("/docs/", ServeUI)
	r.Get("/swagger", redirectDocs)
	r.Get("/swagger/", redirectDocs)
}

func ServeSpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(api.OpenAPI)
}

func ServeUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(api.SwaggerUI)
}

func redirectDocs(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/docs", http.StatusFound)
}
