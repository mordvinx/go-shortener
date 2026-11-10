package middleware

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/mordvinx/go-shortener/internal/handler"
	"github.com/mordvinx/go-shortener/internal/repository"
	"github.com/mordvinx/go-shortener/internal/service"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func gzipBytes(t *testing.T, body string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	if _, err := io.WriteString(writer, body); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func decodedBody(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	if w.Header().Get("Content-Encoding") != "gzip" {
		return w.Body.String()
	}
	reader, err := gzip.NewReader(bytes.NewReader(w.Body.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestGzipRoundTrip(t *testing.T) {
	for _, compressedRequest := range []bool{false, true} {
		for _, compressedResponse := range []bool{false, true} {
			t.Run(strconv.FormatBool(compressedRequest)+"/"+strconv.FormatBool(compressedResponse), func(t *testing.T) {
				h := &handler.Handler{Base: "http://localhost:8080", Shortener: service.NewShortener(repository.NewMemory(), 0)}
				router := Gzip(h.Router())
				const original = "https://example.com/page"
				body := []byte(`{"url":"` + original + `"}`)
				if compressedRequest {
					body = gzipBytes(t, string(body))
				}
				r := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(body))
				r.Header.Set("Content-Type", "application/json")
				if compressedRequest {
					r.Header.Set("Content-Encoding", "gzip")
				}
				if compressedResponse {
					r.Header.Set("Accept-Encoding", "gzip")
				}
				w := httptest.NewRecorder()
				router.ServeHTTP(w, r)
				if w.Code != http.StatusCreated {
					t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
				}
				if got := w.Header().Get("Content-Encoding") == "gzip"; got != compressedResponse {
					t.Fatalf("compressed=%v", got)
				}
				var response struct {
					Result string `json:"result"`
				}
				if err := json.Unmarshal([]byte(decodedBody(t, w)), &response); err != nil {
					t.Fatal(err)
				}
				if !strings.HasPrefix(response.Result, "http://localhost:8080/") {
					t.Fatalf("URL=%q", response.Result)
				}
				redirect := httptest.NewRecorder()
				router.ServeHTTP(redirect, httptest.NewRequest(http.MethodGet, response.Result, nil))
				if redirect.Code != http.StatusTemporaryRedirect || redirect.Header().Get("Location") != original {
					t.Errorf("redirect status=%d location=%q", redirect.Code, redirect.Header().Get("Location"))
				}
			})
		}
	}
}

func TestGzipResponse(t *testing.T) {
	for _, tt := range []struct {
		name, accept, contentType string
		compress                  bool
	}{
		{"json", "gzip", "application/json", true},
		{"html charset", "gzip", "text/html; charset=utf-8", true},
		{"plain text", "gzip", "text/plain", false},
		{"no accept", "", "application/json", false},
		{"other encoding", "br", "application/json", false},
		{"list", "br, gzip", "application/json", true},
		{"disabled", "gzip;q=0", "application/json", false},
		{"quality", "gzip;q=0.5", "application/json", true},
		{"wildcard", "*", "application/json", true},
		{"explicit refusal", "*;q=1, gzip;q=0", "application/json", false},
		{"invalid quality", "gzip;q=wrong", "application/json", false},
		{"not a token", "xgzip", "application/json", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			const body = "test response"
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tt.contentType)
				w.Header().Set("Content-Length", strconv.Itoa(len(body)))
				w.WriteHeader(http.StatusOK)
				io.WriteString(w, body)
			})
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("Accept-Encoding", tt.accept)
			w := httptest.NewRecorder()
			Gzip(next).ServeHTTP(w, r)
			if got := w.Header().Get("Content-Encoding") == "gzip"; got != tt.compress {
				t.Fatalf("compressed=%v", got)
			}
			if got := decodedBody(t, w); got != body {
				t.Errorf("body=%q", got)
			}
			if tt.compress && w.Header().Get("Content-Length") != "" {
				t.Error("stale Content-Length")
			}
			if !strings.Contains(w.Header().Get("Vary"), "Accept-Encoding") {
				t.Error("missing Vary")
			}
		})
	}
}

func TestGzipInvalidRequest(t *testing.T) {
	valid := gzipBytes(t, `{"url":"https://example.com"}`)
	corrupt := append([]byte(nil), valid...)
	corrupt[len(corrupt)-8] ^= 255
	for _, body := range [][]byte{[]byte("not gzip"), valid[:len(valid)-4], corrupt} {
		called := false
		next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
		r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		r.Header.Set("Content-Encoding", "gzip")
		w := httptest.NewRecorder()
		Gzip(next).ServeHTTP(w, r)
		if w.Code != http.StatusBadRequest || called {
			t.Errorf("status=%d handler called=%v", w.Code, called)
		}
	}
}

func TestGzipPlainEndpoint(t *testing.T) {
	h := &handler.Handler{Base: "http://localhost:8080", Shortener: service.NewShortener(repository.NewMemory(), 0)}
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(gzipBytes(t, "https://example.com/")))
	r.Header.Set("Content-Encoding", "gzip")
	r.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	Gzip(h.Router()).ServeHTTP(w, r)
	if w.Code != http.StatusCreated || w.Header().Get("Content-Encoding") != "" || !strings.HasPrefix(w.Body.String(), h.Base+"/") {
		t.Errorf("status=%d headers=%v body=%q", w.Code, w.Header(), w.Body.String())
	}
}

func TestGzipLoggingCountsCompressedBytes(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"result":"`+strings.Repeat("a", 1000)+`"}`)
	})
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	Logging(zap.New(core))(Gzip(next)).ServeHTTP(w, r)
	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("entries=%d", len(entries))
	}
	if got := entries[0].ContextMap()["size"]; got != int64(w.Body.Len()) {
		t.Errorf("logged size=%v actual=%d", got, w.Body.Len())
	}
	decodedBody(t, w) // Includes verification of the gzip trailer written by Close.
}
