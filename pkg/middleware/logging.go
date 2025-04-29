package middleware

import (
	"log"
	"net/http"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wrapped := &responseWriter{ResponseWriter: w, status: 200} // Default to 200 OK
		start := time.Now()

		next.ServeHTTP(wrapped, r)

		log.Printf(
			"[%s] %s - Status: %d - Duration: %v",
			r.Method,
			r.RequestURI,
			wrapped.status,
			time.Since(start),
		)
	})
}
