package handler

import (
	"net/http"
)

func (h *Handler) Resolve(w http.ResponseWriter, r *http.Request) {
	short := r.PathValue("id")

	url, err := h.Shortener.Resolve(short)

	if err != nil {
		http.Error(w, "This link is not valid", http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}
