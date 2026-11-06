package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
)

type shortenRequest struct {
	URL string `json:"url"`
}

type shortenResponse struct {
	Result string `json:"result"`
}

func (h *Handler) ShortenJSON(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	var request shortenRequest
	if err := json.Unmarshal(body, &request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	parsed, err := url.Parse(request.URL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" {
		http.Error(w, "Expected an absolute HTTP or HTTPS URL", http.StatusBadRequest)
		return
	}
	id, err := h.Shortener.Shorten(request.URL)
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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(shortenResponse{Result: shortURL}); err != nil {
		log.Printf("failed to write JSON response: %v", err)
	}
}
