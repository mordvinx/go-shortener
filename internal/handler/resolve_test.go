package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mordvinx/go-shortener/internal/service"
)

func TestResolve(t *testing.T) {
	const originalURL = "https://practicum.yandex.ru/"
	tests := []struct {
		name         string
		id           string
		wantStatus   int
		wantLocation string
	}{
		{"known ID", "known", http.StatusTemporaryRedirect, originalURL},
		{"unknown ID", "unknown", http.StatusBadRequest, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{Shortener: &service.Shortener{
				Store: map[string]string{"known": originalURL},
			}}
			r := httptest.NewRequest(http.MethodGet, "/"+tt.id, nil)
			// При прямом вызове хендлера mux не заполняет параметр пути.
			r.SetPathValue("id", tt.id)
			w := httptest.NewRecorder()

			h.Resolve(w, r)

			if w.Code != tt.wantStatus {
				t.Errorf("Want response status to be %d, got %d", tt.wantStatus, w.Code)
			}

			gotLocation := w.Header().Get("Location")
			if gotLocation != tt.wantLocation {
				t.Errorf("Want Location header to be %q, got %q", tt.wantLocation, gotLocation)
			}
		})
	}
}
