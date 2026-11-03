package middleware

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestLogging(t *testing.T) {
	for _, tt := range []struct {
		name           string
		handler        http.HandlerFunc
		status         int
		body, location string
	}{
		{"explicit status", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(201); io.WriteString(w, "created") }, 201, "created", ""},
		{"implicit status", func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, "hello") }, 200, "hello", ""},
		{"empty handler", func(w http.ResponseWriter, r *http.Request) {}, 200, "", ""},
		{"multiple writes", func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, "привет")
			io.WriteString(w, "!")
		}, 200, "привет!", ""},
		{"repeated status", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(201); w.WriteHeader(500) }, 201, "", ""},
		{"redirect", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "https://example.com/", 307) }, 307, "<a href=\"https://example.com/\">Temporary Redirect</a>.\n\n", "https://example.com/"},
		{"error", func(w http.ResponseWriter, r *http.Request) { http.Error(w, "invalid", 400) }, 400, "invalid\n", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			core, logs := observer.New(zap.DebugLevel)
			wrapped := Logging(zap.New(core))(tt.handler)
			// Reuse the middleware to verify per-request counters.
			for i := 0; i < 2; i++ {
				r := httptest.NewRequest(http.MethodGet, "/test?x=1", nil)
				w := httptest.NewRecorder()
				start := time.Now()
				wrapped.ServeHTTP(w, r)
				elapsed := time.Since(start)
				if w.Code != tt.status || w.Body.String() != tt.body || w.Header().Get("Location") != tt.location {
					t.Fatalf("response: status=%d body=%q location=%q", w.Code, w.Body.String(), w.Header().Get("Location"))
				}
				entries := logs.TakeAll()
				if len(entries) != 1 {
					t.Fatalf("got %d log entries, want 1", len(entries))
				}
				entry := entries[0]
				if entry.Level != zap.InfoLevel {
					t.Errorf("level=%v, want Info", entry.Level)
				}
				fields := entry.ContextMap()
				for key, want := range map[string]any{"uri": "/test?x=1", "method": "GET", "status": int64(tt.status), "size": int64(len(tt.body))} {
					if fields[key] != want {
						t.Errorf("%s=%v, want %v", key, fields[key], want)
					}
				}
				duration, ok := fields["duration"].(time.Duration)
				if !ok || duration < 0 || duration > elapsed {
					t.Errorf("invalid duration: %v", fields["duration"])
				}
			}
		})
	}
}

type writerDouble struct {
	header   http.Header
	statuses []int
	n        int
	err      error
}

func (w *writerDouble) Header() http.Header       { return w.header }
func (w *writerDouble) WriteHeader(status int)    { w.statuses = append(w.statuses, status) }
func (w *writerDouble) Write([]byte) (int, error) { return w.n, w.err }

func TestResponseWriterPartialWrite(t *testing.T) {
	wantErr := errors.New("write failed")
	underlying := &writerDouble{header: make(http.Header), n: 2, err: wantErr}
	w := &responseWriter{ResponseWriter: underlying}
	n, err := w.Write([]byte("hello"))
	if n != 2 || !errors.Is(err, wantErr) || w.size != 2 || w.status != 200 {
		t.Errorf("n=%d err=%v size=%d status=%d", n, err, w.size, w.status)
	}
	if len(underlying.statuses) != 1 || underlying.statuses[0] != 200 {
		t.Errorf("statuses=%v", underlying.statuses)
	}
}

func TestResponseWriterForwardsStatuses(t *testing.T) {
	underlying := &writerDouble{header: make(http.Header)}
	w := &responseWriter{ResponseWriter: underlying}
	w.WriteHeader(103)
	if w.status != 0 {
		t.Fatalf("informational response finalized status: %d", w.status)
	}
	w.WriteHeader(201)
	w.WriteHeader(500)
	if w.status != 201 {
		t.Errorf("status=%d, want 201", w.status)
	}
	want := []int{103, 201, 500}
	if len(underlying.statuses) != len(want) {
		t.Fatalf("statuses=%v", underlying.statuses)
	}
	for i, status := range want {
		if underlying.statuses[i] != status {
			t.Errorf("statuses=%v", underlying.statuses)
		}
	}
}
