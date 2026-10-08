package handler

import (
	"io"
	"net/http"
	"net/url"
)

func (h *Handler) Shorten(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)

	if err != nil {
		http.Error(w, "Broken request body", http.StatusBadRequest)
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
		http.Error(w, "Error creating short URL", http.StatusInternalServerError)
		return
	}

	shortURL := "http://localhost:8080/" + id

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)

	io.WriteString(w, shortURL)
}
