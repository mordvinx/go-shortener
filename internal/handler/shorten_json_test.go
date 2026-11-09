package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/mordvinx/go-shortener/internal/repository"
	"github.com/mordvinx/go-shortener/internal/service"
)

func TestShortenJSONRoundTrip(t *testing.T) {
	for _, base := range []string{"http://localhost:8080", "http://localhost:8080/"} {
		t.Run(base, func(t *testing.T) {
			h := &Handler{Base: base, Shortener: service.NewShortener(repository.NewMemory())}
			router := h.Router()
			const original = "https://example.com/path?a=1&b=2"
			r := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(`{"url":"`+original+`"}`))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			if w.Code != http.StatusCreated {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			if got := w.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type=%q", got)
			}
			var response map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			short := response["result"]
			if len(response) != 1 || !strings.HasPrefix(short, "http://localhost:8080/") {
				t.Fatalf("response=%v", response)
			}
			u, err := url.Parse(short)
			if err != nil || u.Path == "/" {
				t.Fatalf("invalid short URL %q", short)
			}
			resolved := httptest.NewRecorder()
			router.ServeHTTP(resolved, httptest.NewRequest(http.MethodGet, u.Path, nil))
			if resolved.Code != http.StatusTemporaryRedirect || resolved.Header().Get("Location") != original {
				t.Errorf("resolve status=%d Location=%q", resolved.Code, resolved.Header().Get("Location"))
			}
		})
	}
}

func TestShortenJSONInvalidRequest(t *testing.T) {
	for _, body := range []string{"", `{`, `{}`, `null`, `[]`, `{"url":42}`, `{"url":""}`, `{"url":"relative"}`, `{"url":"ftp://example.com"}`, `{"url":"https:///path"}`, `{"url":"https://example.com/%zz"}`, `{"url":"https://example.com"} {}`} {
		t.Run(body, func(t *testing.T) {
			s := &recordingShortener{}
			h := &Handler{Base: "http://localhost:8080", Shortener: s}
			w := httptest.NewRecorder()
			h.Router().ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body)))
			if w.Code != http.StatusBadRequest {
				t.Errorf("status=%d, want 400", w.Code)
			}
			if s.called {
				t.Error("invalid request called service")
			}
		})
	}
}

func TestShortenJSONInternalErrors(t *testing.T) {
	for _, name := range []string{"read body", "service", "base URL"} {
		t.Run(name, func(t *testing.T) {
			s := &recordingShortener{}
			h := &Handler{Base: "http://localhost:8080", Shortener: s}
			r := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(`{"url":"https://example.com/"}`))
			switch name {
			case "read body":
				r.Body = io.NopCloser(iotest.ErrReader(errors.New("read failed")))
			case "service":
				s.err = errors.New("service failed")
			case "base URL":
				h.Base = "http://example.com/%zz"
			}
			w := httptest.NewRecorder()
			h.ShortenJSON(w, r)
			if w.Code != http.StatusInternalServerError || w.Body.String() != "Internal Server Error\n" {
				t.Errorf("status=%d body=%q", w.Code, w.Body.String())
			}
			if name == "read body" && s.called {
				t.Error("service called after read failure")
			}
		})
	}
}
