package middleware

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/madmike/go-infra/telemetry"
)

// RequestLogger logs HTTP requests
func RequestLogger(logger telemetry.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Generate request ID
			requestID := uuid.New().String()
			r.Header.Set("X-Request-ID", requestID)
			w.Header().Set("X-Request-ID", requestID)

			// Wrap response writer to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Call next handler
			next.ServeHTTP(wrapped, r)

			// Log request
			duration := time.Since(start)
			logger.Info("HTTP request",
				telemetry.String("method", r.Method),
				telemetry.String("path", r.URL.Path),
				telemetry.String("request_id", requestID),
				telemetry.Int("status", wrapped.statusCode),
				telemetry.Duration("duration", duration),
				telemetry.String("remote_addr", r.RemoteAddr),
			)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, fmt.Errorf("http.Hijacker not implemented")
}
