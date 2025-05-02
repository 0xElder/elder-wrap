package middleware

import (
	"net/http"
	"time"

	"github.com/0xElder/elder-wrap/pkg/logging"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func RestLoggingMiddleware(next http.Handler, logger logging.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wrapped := &responseWriter{ResponseWriter: w, status: 200} // Default to 200 OK
		start := time.Now()

		next.ServeHTTP(wrapped, r)
		logger.Info(r.Context(), "Request handled",
			"method", r.Method,
			"uri", r.RequestURI,
			"status", wrapped.status,
			"duration", time.Since(start),
		)
	})
}
