package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// responseWriter forwards the response and records its status and body size.
type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (w *responseWriter) WriteHeader(status int) {
	if w.status == 0 && (status == http.StatusSwitchingProtocols || status >= 200) {
		w.status = status
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}

	n, err := w.ResponseWriter.Write(body)

	w.size += n

	return n, err
}

func Logging(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			wrapped := &responseWriter{
				ResponseWriter: w,
			}

			next.ServeHTTP(wrapped, r)

			if wrapped.status == 0 {
				wrapped.status = http.StatusOK
			}

			logger.Info(
				"Logging middleware",
				zap.String("uri", r.RequestURI),
				zap.String("method", r.Method),
				zap.Duration("duration", time.Since(start)),
				zap.Int("status", wrapped.status),
				zap.Int("size", wrapped.size),
			)
		})
	}
}
