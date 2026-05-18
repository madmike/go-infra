package middleware

import (
	"net/http"

	"github.com/madmike/go-infra/telemetry"
)

// Recovery recovers from panics and logs them
func Recovery(logger telemetry.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("Panic recovered",
						telemetry.Any("error", err),
						telemetry.String("method", r.Method),
						telemetry.String("path", r.URL.Path),
					)

					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"success":false,"error":"Internal server error"}`))
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
