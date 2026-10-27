package docs

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestSpecAndUI(t *testing.T) {
	t.Parallel()

	r := chi.NewRouter()
	Mount(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)

	spec := get(t, srv, "/openapi.yaml")
	if spec.StatusCode != http.StatusOK {
		t.Fatalf("spec status = %d", spec.StatusCode)
	}
	if !strings.Contains(spec.Body, "Digital Lending Platform API") {
		t.Fatalf("spec missing title: %s", spec.Body)
	}
	if !strings.Contains(spec.Header.Get("Content-Type"), "yaml") {
		t.Fatalf("spec content-type = %s", spec.Header.Get("Content-Type"))
	}

	ui := get(t, srv, "/docs")
	if ui.StatusCode != http.StatusOK {
		t.Fatalf("docs status = %d", ui.StatusCode)
	}
	if !strings.Contains(ui.Body, "swagger-ui") {
		t.Fatalf("docs missing swagger-ui: %s", ui.Body)
	}

	redir := getNoFollow(t, srv, "/swagger")
	if redir.StatusCode != http.StatusFound {
		t.Fatalf("swagger redirect status = %d", redir.StatusCode)
	}
	if loc := redir.Header.Get("Location"); loc != "/docs" {
		t.Fatalf("swagger redirect location = %s", loc)
	}
}

type resp struct {
	StatusCode int
	Header     http.Header
	Body       string
}

func get(t *testing.T, srv *httptest.Server, path string) resp {
	t.Helper()
	res, err := http.Get(srv.URL + path)
	if err != nil {
		t.Fatalf("get %s: %v", path, err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return resp{StatusCode: res.StatusCode, Header: res.Header, Body: string(body)}
}

func getNoFollow(t *testing.T, srv *httptest.Server, path string) resp {
	t.Helper()
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	res, err := client.Get(srv.URL + path)
	if err != nil {
		t.Fatalf("get %s: %v", path, err)
	}
	defer res.Body.Close()
	return resp{StatusCode: res.StatusCode, Header: res.Header}
}
