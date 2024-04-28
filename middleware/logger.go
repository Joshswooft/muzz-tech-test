package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type WrappedWriter struct {
	http.ResponseWriter
	statusCode int
}

// Implement the http.ResponseWriter interface
func (w *WrappedWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
	w.statusCode = statusCode
}

// logs the http requests
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &WrappedWriter{statusCode: http.StatusOK, ResponseWriter: w}

		next.ServeHTTP(wrapped, r)
		slog.Info("request", slog.Int("code", wrapped.statusCode), slog.String("method", r.Method), slog.String("path", r.URL.Path), slog.Int64("durationMS", time.Since(start).Milliseconds()))
	})
}
