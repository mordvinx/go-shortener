package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"log"
	"mime"
	"net/http"
	"strconv"
	"strings"
)

// Gzip decodes gzip requests and compresses JSON and HTML responses when accepted.
func Gzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.EqualFold(strings.TrimSpace(r.Header.Get("Content-Encoding")), "gzip") {
			reader, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Invalid gzip body", http.StatusBadRequest)
				return
			}
			body, err := io.ReadAll(reader)
			reader.Close()
			if err != nil {
				http.Error(w, "Invalid gzip body", http.StatusBadRequest)
				return
			}
			r.Body.Close()
			r.Body = io.NopCloser(bytes.NewReader(body))
			r.ContentLength = int64(len(body))
			r.Header.Del("Content-Encoding")
			r.Header.Del("Content-Length")
		}

		// Responses can differ by Accept-Encoding, including the uncompressed variant.
		w.Header().Add("Vary", "Accept-Encoding")
		if !acceptsGzip(r.Header.Values("Accept-Encoding")) || r.Method == http.MethodHead {
			next.ServeHTTP(w, r)
			return
		}
		wrapped := &gzipResponseWriter{ResponseWriter: w}
		defer func() {
			if wrapped.writer != nil {
				if err := wrapped.writer.Close(); err != nil {
					log.Printf("failed to close gzip response: %v", err)
				}
			}
		}()
		next.ServeHTTP(wrapped, r)
	})
}

func acceptsGzip(values []string) bool {
	wildcard := false
	for _, value := range values {
		for _, item := range strings.Split(value, ",") {
			parts := strings.Split(item, ";")
			name := strings.ToLower(strings.TrimSpace(parts[0]))
			if name != "gzip" && name != "*" {
				continue
			}
			quality := 1.0
			for _, parameter := range parts[1:] {
				key, value, ok := strings.Cut(strings.TrimSpace(parameter), "=")
				if ok && strings.EqualFold(key, "q") {
					parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
					if err != nil || !(parsed >= 0 && parsed <= 1) {
						quality = 0
					} else {
						quality = parsed
					}
				}
			}
			if name == "gzip" {
				return quality > 0
			}
			wildcard = quality > 0
		}
	}
	return wildcard
}

type gzipResponseWriter struct {
	http.ResponseWriter
	writer      *gzip.Writer
	wroteHeader bool
}

func (w *gzipResponseWriter) WriteHeader(status int) {
	if w.wroteHeader || (status >= 100 && status < 200 && status != http.StatusSwitchingProtocols) {
		w.ResponseWriter.WriteHeader(status)
		return
	}
	w.wroteHeader = true
	contentType, _, _ := mime.ParseMediaType(w.Header().Get("Content-Type"))
	if (contentType == "application/json" || contentType == "text/html") &&
		w.Header().Get("Content-Encoding") == "" && status >= 200 &&
		status != http.StatusNoContent && status != http.StatusNotModified {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length")
		w.writer = gzip.NewWriter(w.ResponseWriter)
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *gzipResponseWriter) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", http.DetectContentType(body))
		}
		w.WriteHeader(http.StatusOK)
	}
	if w.writer != nil {
		return w.writer.Write(body)
	}
	return w.ResponseWriter.Write(body)
}
