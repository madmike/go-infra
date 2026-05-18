package telemetry

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// ContextKey represents a key for context values
type ContextKey string

const (
	// ContextKeyRequestID is the key for request ID in context
	ContextKeyRequestID ContextKey = "request_id"
	// ContextKeySessionID is the key for session ID in context
	ContextKeySessionID ContextKey = "session_id"
	// ContextKeyUserID is the key for user ID in context
	ContextKeyUserID ContextKey = "user_id"
	// ContextKeyProviderID is the key for provider ID in context
	ContextKeyProviderID ContextKey = "provider_id"
	// ContextKeyCapability is the key for capability in context
	ContextKeyCapability ContextKey = "capability"
)

// Logger defines the interface for structured logging
type Logger interface {
	// Trace logs a trace message (most verbose level)
	Trace(msg string, fields ...Field)

	// Debug logs a debug message
	Debug(msg string, fields ...Field)

	// Info logs an info message
	Info(msg string, fields ...Field)

	// Warn logs a warning message
	Warn(msg string, fields ...Field)

	// Error logs an error message
	Error(msg string, fields ...Field)

	// Fatal logs a fatal message and exits
	Fatal(msg string, fields ...Field)

	// WithContext returns a logger with context values
	WithContext(ctx context.Context) Logger

	// WithFields returns a logger with additional fields
	WithFields(fields ...Field) Logger

	// WithModule returns a logger with a module name
	WithModule(module string) Logger
}

// NoOpLogger is a logger that does nothing (useful for optional logging)
type NoOpLogger struct{}

func (l *NoOpLogger) Trace(msg string, fields ...Field)      {}
func (l *NoOpLogger) Debug(msg string, fields ...Field)      {}
func (l *NoOpLogger) Info(msg string, fields ...Field)       {}
func (l *NoOpLogger) Warn(msg string, fields ...Field)       {}
func (l *NoOpLogger) Error(msg string, fields ...Field)      {}
func (l *NoOpLogger) Fatal(msg string, fields ...Field)      {}
func (l *NoOpLogger) WithContext(ctx context.Context) Logger { return l }
func (l *NoOpLogger) WithFields(fields ...Field) Logger      { return l }
func (l *NoOpLogger) WithModule(module string) Logger        { return l }

// Field represents a structured log field
type Field struct {
	Key   string
	Value any
}

// String creates a string field
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int creates an int field
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Int64 creates an int64 field
func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

// Float64 creates a float64 field
func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

// Bool creates a bool field
func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// Duration creates a duration field
func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value}
}

// Time creates a time field
func Time(key string, value time.Time) Field {
	return Field{Key: key, Value: value}
}

// Err creates an error field
func Err(err error) Field {
	return Field{Key: "error", Value: err}
}

// Any creates a field with any value
func Any(key string, value any) Field {
	return Field{Key: key, Value: value}
}

// zerologLogger implements Logger using zerolog
type zerologLogger struct {
	logger zerolog.Logger
}

// Config contains configuration for the logger
type Config struct {
	// Level is the minimum log level (debug, info, warn, error, fatal)
	Level string

	// Format is the log format (json, console)
	Format string

	// EnableCaller enables caller information in logs
	EnableCaller bool

	// ServiceName is the name of the service
	ServiceName string

	// Environment is the deployment environment (dev, staging, prod)
	Environment string
}

// New creates a new Logger instance
func New(config Config) Logger {
	// Configure zerolog
	zerolog.TimeFieldFormat = time.RFC3339Nano

	// Set up output writer based on format
	var output io.Writer = os.Stdout

	// Use module console writer for human-readable output with module field
	switch config.Format {
	case "console", "text":
		output = &ModuleConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
			NoColor:    false,
		}
	case "json":
		// Explicitly use JSON format
		output = os.Stdout
	}

	// Create base logger
	logger := zerolog.New(output).With().Timestamp().Logger()

	// Set log level
	level := parseLogLevel(config.Level)
	logger = logger.Level(level)

	// Add caller information if enabled
	if config.EnableCaller {
		// Skip frames to show the actual caller, taking into account our wrapper
		logger = logger.With().CallerWithSkipFrameCount(zerolog.CallerSkipFrameCount + 1).Logger()
	}

	// Add environment if provided
	if config.Environment != "" {
		logger = logger.With().Str("environment", config.Environment).Logger()
	}

	return &zerologLogger{logger: logger}
}

// Trace logs a trace message
func (l *zerologLogger) Trace(msg string, fields ...Field) {
	event := l.logger.Trace()
	l.addFields(event, fields)
	event.Msg(msg)
}

// Debug logs a debug message
func (l *zerologLogger) Debug(msg string, fields ...Field) {
	event := l.logger.Debug()
	l.addFields(event, fields)
	event.Msg(msg)
}

// Info logs an info message
func (l *zerologLogger) Info(msg string, fields ...Field) {
	event := l.logger.Info()
	l.addFields(event, fields)
	event.Msg(msg)
}

// Warn logs a warning message
func (l *zerologLogger) Warn(msg string, fields ...Field) {
	event := l.logger.Warn()
	l.addFields(event, fields)
	event.Msg(msg)
}

// Error logs an error message
func (l *zerologLogger) Error(msg string, fields ...Field) {
	event := l.logger.Error()
	l.addFields(event, fields)
	event.Msg(msg)
}

// Fatal logs a fatal message and exits
func (l *zerologLogger) Fatal(msg string, fields ...Field) {
	event := l.logger.Fatal()
	l.addFields(event, fields)
	event.Msg(msg)
}

// WithContext returns a logger with context values
func (l *zerologLogger) WithContext(ctx context.Context) Logger {
	logger := l.logger

	// Extract correlation IDs from context
	if requestID := ctx.Value(ContextKeyRequestID); requestID != nil {
		if id, ok := requestID.(string); ok {
			logger = logger.With().Str("request_id", id).Logger()
		}
	}

	if sessionID := ctx.Value(ContextKeySessionID); sessionID != nil {
		if id, ok := sessionID.(string); ok {
			logger = logger.With().Str("session_id", id).Logger()
		}
	}

	if userID := ctx.Value(ContextKeyUserID); userID != nil {
		if id, ok := userID.(string); ok {
			logger = logger.With().Str("user_id", id).Logger()
		}
	}

	if providerID := ctx.Value(ContextKeyProviderID); providerID != nil {
		if id, ok := providerID.(string); ok {
			logger = logger.With().Str("provider_id", id).Logger()
		}
	}

	if capability := ctx.Value(ContextKeyCapability); capability != nil {
		if cap, ok := capability.(string); ok {
			logger = logger.With().Str("capability", cap).Logger()
		}
	}

	return &zerologLogger{logger: logger}
}

// WithFields returns a logger with additional fields
func (l *zerologLogger) WithFields(fields ...Field) Logger {
	logger := l.logger
	for _, field := range fields {
		logger = logger.With().Interface(field.Key, field.Value).Logger()
	}
	return &zerologLogger{logger: logger}
}

// WithModule returns a logger with a module name
func (l *zerologLogger) WithModule(module string) Logger {
	logger := l.logger.With().Str("module", module).Logger()
	return &zerologLogger{logger: logger}
}

// addFields adds fields to a zerolog event
func (l *zerologLogger) addFields(event *zerolog.Event, fields []Field) {
	for _, field := range fields {
		switch v := field.Value.(type) {
		case string:
			event.Str(field.Key, v)
		case int:
			event.Int(field.Key, v)
		case int64:
			event.Int64(field.Key, v)
		case float64:
			event.Float64(field.Key, v)
		case bool:
			event.Bool(field.Key, v)
		case time.Duration:
			event.Dur(field.Key, v)
		case time.Time:
			event.Time(field.Key, v)
		case error:
			event.Err(v)
		default:
			event.Interface(field.Key, v)
		}
	}
}

// parseLogLevel parses a log level string to zerolog.Level
func parseLogLevel(level string) zerolog.Level {
	switch level {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}

// Default creates a logger with default configuration
func Default() Logger {
	return New(Config{
		Level:        "info",
		Format:       "json",
		EnableCaller: false,
		ServiceName:  "placeholder",
		Environment:  "development",
	})
}

// ContextWithRequestID adds a request ID to the context
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, ContextKeyRequestID, requestID)
}

// ContextWithSessionID adds a session ID to the context
func ContextWithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, ContextKeySessionID, sessionID)
}

// ContextWithUserID adds a user ID to the context
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, ContextKeyUserID, userID)
}

// ContextWithProviderID adds a provider ID to the context
func ContextWithProviderID(ctx context.Context, providerID string) context.Context {
	return context.WithValue(ctx, ContextKeyProviderID, providerID)
}

// ContextWithCapability adds a capability to the context
func ContextWithCapability(ctx context.Context, capability string) context.Context {
	return context.WithValue(ctx, ContextKeyCapability, capability)
}

// GetRequestIDFromContext extracts the request ID from context
func GetRequestIDFromContext(ctx context.Context) string {
	if requestID := ctx.Value(ContextKeyRequestID); requestID != nil {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}

// GetSessionIDFromContext extracts the session ID from context
func GetSessionIDFromContext(ctx context.Context) string {
	if sessionID := ctx.Value(ContextKeySessionID); sessionID != nil {
		if id, ok := sessionID.(string); ok {
			return id
		}
	}
	return ""
}
