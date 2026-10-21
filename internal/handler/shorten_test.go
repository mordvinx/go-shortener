package handler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/mordvinx/go-shortener/internal/service"
)

func TestShorten(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"valid URL", "https://practicum.yandex.ru/", http.StatusCreated},
		{"empty body", "", http.StatusBadRequest},
		{"relative URL", "hello", http.StatusBadRequest},
		{"invalid escape", "https://example.com/%zz", http.StatusBadRequest},
		{"missing host", "https:///path", http.StatusBadRequest},
		{"unsupported scheme", "ftp://example.com/", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{Shortener: &service.Shortener{}}
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			r.Header.Set("Content-Type", "text/plain")
			w := httptest.NewRecorder()

			h.Shorten(w, r)

			if w.Code != tt.wantStatus {
				t.Errorf("Want response status to be %d, got %d", tt.wantStatus, w.Code)
			}

			if tt.wantStatus != http.StatusCreated {
				return
			}

			wantContentType := "text/plain"
			gotContentType := w.Header().Get("Content-Type")

			if gotContentType != wantContentType {
				t.Errorf("Want Content-Type header to be %q, got %q", wantContentType, gotContentType)
			}

			u, err := url.Parse(w.Body.String())

			if err != nil {
				t.Fatalf("Response is not a valid URL: %q (%s)", w.Body.String(), err)
			}

			if u.Host != "localhost:8080" {
				t.Errorf("Invalid host (or changed): %q", u.Host)
			}

			if u.Scheme != "http" {
				t.Errorf("Invalid scheme (or changed): %q", u.Scheme)
			}

			id := strings.TrimPrefix(u.Path, "/")

			if id == "" {
				t.Fatalf("Server responded with empty id")
			}

			gotURL, errR := h.Shortener.Resolve(id)

			if errR != nil {
				t.Fatalf("Can not resolve shortened id %q", id)
			}

			if gotURL != tt.body {
				t.Errorf("Want original URL to be %q, got %q", tt.body, gotURL)
			}
		})
	}
}
