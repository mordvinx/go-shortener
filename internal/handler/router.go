package handler

import (
	"net/http"
	"path"
)

func (h *Handler) Router() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /{$}", h.Shorten)

	mux.HandleFunc("GET /{id}", h.Resolve)
	mux.HandleFunc("HEAD /{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Invalid request", http.StatusBadRequest)
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Reject non-canonical paths before ServeMux can redirect them.
		if path.Clean(r.URL.Path) != r.URL.Path {
			http.Error(w, "Invalid request path", http.StatusBadRequest)
			return
		}
		mux.ServeHTTP(w, r)
	})
}
