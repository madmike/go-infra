package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/madmike/go-infra/telemetry"
)

// GinLogger returns a gin.HandlerFunc that logs requests using telemetry.Logger
func GinLogger(logger telemetry.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Generate or use existing request ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)

		// Process request
		c.Next()

		// Log request
		end := time.Now()
		latency := end.Sub(start)

		fields := []telemetry.Field{
			telemetry.Int("status", c.Writer.Status()),
			telemetry.String("method", c.Request.Method),
			telemetry.String("path", path),
			telemetry.String("query", query),
			telemetry.String("ip", c.ClientIP()),
			telemetry.String("user_agent", c.Request.UserAgent()),
			telemetry.Duration("latency", latency),
			telemetry.String("request_id", requestID),
		}
		switch {
		case c.Writer.Status() >= 500:
			if len(c.Errors) > 0 {
				for _, e := range c.Errors.Errors() {
					logger.Error(e, append(fields, telemetry.String("error", e))...)
				}
			} else {
				logger.Error("HTTP Request", fields...)
			}
		case c.Writer.Status() >= 400:
			if len(c.Errors) > 0 {
				for _, e := range c.Errors.Errors() {
					logger.Warn(e, append(fields, telemetry.String("error", e))...)
				}
			} else {
				logger.Warn("HTTP Request", fields...)
			}
		default:
			logger.Info("HTTP Request", fields...)
		}
	}
}

// GinRecovery returns a gin.HandlerFunc that recovers from panics and logs them
func GinRecovery(logger telemetry.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("Panic recovered",
					telemetry.Any("error", err),
					telemetry.String("path", c.Request.URL.Path),
					telemetry.String("request_id", c.GetString("request_id")),
				)
				c.AbortWithStatus(500)
			}
		}()
		c.Next()
	}
}
