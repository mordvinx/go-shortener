package handler

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/mordvinx/go-shortener/internal/repository"
	"github.com/mordvinx/go-shortener/internal/service"
)

type recordingShortener struct {
	called bool
	err    error
}

func (s *recordingShortener) Shorten(string) (string, error) {
	s.called = true
	if s.err != nil {
		return "", s.err
	}
	return "test-id", nil
}

func (s *recordingShortener) Resolve(string) (string, error) {
	return "", nil
}

func TestShortenBodyReadError(t *testing.T) {
	s := &recordingShortener{}
	h := &Handler{Base: "http://localhost:8080", Shortener: s}
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Body = io.NopCloser(iotest.ErrReader(errors.New("body read failed")))
	w := httptest.NewRecorder()

	h.Shorten(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("got status %d, want 500", w.Code)
	}
	if got, want := w.Body.String(), "Internal Server Error\n"; got != want {
		t.Errorf("got body %q, want %q", got, want)
	}
	if s.called {
		t.Error("Shorten must not be called when reading the body fails")
	}
}

func TestShortenServiceError(t *testing.T) {
	serviceErr := errors.New("ID generation failed")
	s := &recordingShortener{err: serviceErr}
	h := &Handler{Base: "http://localhost:8080", Shortener: s}
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com/"))
	w := httptest.NewRecorder()

	// The standard logger is shared, so this test must not run in parallel.
	var logs bytes.Buffer
	previousOutput := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(previousOutput) })

	h.Shorten(w, r)

	if !s.called {
		t.Error("expected Shorten to be called")
		return
	}
	if w.Code != http.StatusInternalServerError {
		t.Errorf("got status %d, want 500", w.Code)
	}
	if got, want := w.Body.String(), "Internal Server Error\n"; got != want {
		t.Errorf("got body %q, want %q", got, want)
	}
	if !strings.Contains(logs.String(), serviceErr.Error()) {
		t.Errorf("log %q does not contain service error %q", logs.String(), serviceErr)
	}
}

func TestShortenBaseURL(t *testing.T) {
	for _, tt := range []struct {
		name       string
		base       string
		wantStatus int
		wantBody   string
	}{
		{"without trailing slash", "https://example.com/s", http.StatusCreated, "https://example.com/s/test-id"},
		{"with trailing slash", "https://example.com/s/", http.StatusCreated, "https://example.com/s/test-id"},
		{"with query", "https://example.com/s/?source=test", http.StatusCreated, "https://example.com/s/test-id?source=test"},
		{"invalid base", "http://example.com/%zz", http.StatusInternalServerError, "Internal Server Error\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{Base: tt.base, Shortener: &recordingShortener{}}
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://practicum.yandex.ru/"))
			w := httptest.NewRecorder()

			h.Shorten(w, r)

			if w.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", w.Code, tt.wantStatus)
				return
			}
			if got := w.Body.String(); got != tt.wantBody {
				t.Errorf("got body %q, want %q", got, tt.wantBody)
			}
		})
	}
}

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
			h := &Handler{Base: "http://localhost:8080", Shortener: service.NewShortener(repository.NewMemory(), 0)}
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			r.Header.Set("Content-Type", "text/plain")
			w := httptest.NewRecorder()

			h.Shorten(w, r)

			if w.Code != tt.wantStatus {
				t.Errorf("Want response status to be %d, got %d", tt.wantStatus, w.Code)
				return
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
