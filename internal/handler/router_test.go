package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mordvinx/go-shortener/internal/service"
)

func TestShortenAndResolve(t *testing.T) {
	h := &Handler{Shortener: &service.Shortener{}}
	router := h.Router()
	originals := []string{"https://practicum.yandex.ru/", "https://example.com/path?q=value"}
	links := make(map[string]bool)
	for _, original := range originals {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(original))
		req.Header.Set("Content-Type", "text/plain")
		created := httptest.NewRecorder()
		router.ServeHTTP(created, req)
		if created.Code != http.StatusCreated || created.Header().Get("Content-Type") != "text/plain" {
			t.Fatalf("POST: status=%d headers=%v", created.Code, created.Header())
		}
		shortURL := created.Body.String()
		if !strings.HasPrefix(shortURL, "http://localhost:8080/") || links[shortURL] {
			t.Fatalf("invalid or duplicate short URL: %q", shortURL)
		}
		links[shortURL] = true
		resolved := httptest.NewRecorder()
		router.ServeHTTP(resolved, httptest.NewRequest(http.MethodGet, shortURL, nil))
		if resolved.Code != http.StatusTemporaryRedirect || resolved.Header().Get("Location") != original {
			t.Fatalf("GET: status=%d Location=%q", resolved.Code, resolved.Header().Get("Location"))
		}
	}
}

func TestInvalidRequests(t *testing.T) {
	h := &Handler{Shortener: &service.Shortener{}}
	router := h.Router()
	for _, tc := range []struct {
		name, method, target, body string
	}{
		{"empty body", "POST", "/", ""},
		{"relative URL", "POST", "/", "hello"},
		{"invalid escape", "POST", "/", "https://example.com/%zz"},
		{"missing host", "POST", "/", "https:///path"},
		{"unsupported scheme", "POST", "/", "ftp://example.com/"},
		{"unknown ID", "GET", "/unknown", ""},
		{"missing ID", "GET", "/", ""},
		{"wrong POST path", "POST", "/abc", "https://example.com/"},
		{"HEAD", "HEAD", "/abc", ""},
		{"wrong method", "PUT", "/abc", ""},
		{"nested path", "GET", "/abc/def", ""},
		{"double slash", "GET", "//abc", ""},
		{"dot segment", "GET", "/a/../abc", ""},
		{"trailing slash", "GET", "/abc/", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(tc.method, tc.target, strings.NewReader(tc.body)))
			if w.Code != http.StatusBadRequest {
				t.Fatalf("got status %d, want 400", w.Code)
			}
		})
	}
}
