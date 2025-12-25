package middleware

import (
	"log"
	"net/http"
	"time"
)

// LoggingMiddleware logs the incoming HTTP request & its duration.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap the ResponseWriter to capture the status code
		wrappedWriter := &responseWriterWrapper{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrappedWriter, r)

		duration := time.Since(start)
		log.Printf(
			"Method: %s | URL: %s | RemoteAddr: %s | Status: %d | Duration: %v",
			r.Method,
			r.RequestURI,
			r.RemoteAddr,
			wrappedWriter.statusCode,
			duration,
		)
	})
}

// responseWriterWrapper wraps http.ResponseWriter to capture the status code.
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
