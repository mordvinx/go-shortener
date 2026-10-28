package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) Router() http.Handler {
	mux := chi.NewRouter()

	mux.Post("/", h.Shorten)

	mux.Get("/{id}", h.Resolve)

	badRequest := func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Invalid request", http.StatusBadRequest)
	}

	mux.NotFound(badRequest)
	mux.MethodNotAllowed(badRequest)

	return mux
}
