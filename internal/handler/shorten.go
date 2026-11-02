package handler

import (
	"io"
	"log"
	"net/http"
	"net/url"
)

func (h *Handler) Shorten(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)

	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if len(body) == 0 {
		http.Error(w, "URL not provided", http.StatusBadRequest)
		return
	}

	parsed, err := url.Parse(string(body))
	if err != nil {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" {
		http.Error(w, "Expected an absolute HTTP or HTTPS URL", http.StatusBadRequest)
		return
	}

	id, err := h.Shortener.Shorten(string(body))

	if err != nil {
		log.Printf("failed to shorten URL: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	shortURL, err := url.JoinPath(h.Base, id)
	if err != nil {
		log.Printf("failed to build short URL: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)

	io.WriteString(w, shortURL)
}
