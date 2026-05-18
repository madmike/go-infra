package telemetry

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/rs/zerolog"
)

// LogWriter implements io.Writer and redirects output to a logger
type LogWriter struct {
	Logger Logger
	Level  string // info, error, debug
}

func (w *LogWriter) Write(p []byte) (n int, err error) {
	msg := strings.TrimSpace(string(p))
	if msg == "" {
		return len(p), nil
	}

	switch w.Level {
	case "error":
		w.Logger.Error(msg)
	case "debug":
		w.Logger.Debug(msg)
	case "warn":
		w.Logger.Warn(msg)
	default:
		w.Logger.Info(msg)
	}
	return len(p), nil
}

// ModuleConsoleWriter wraps zerolog.ConsoleWriter to inject module field into output
type ModuleConsoleWriter struct {
	Out        io.Writer
	TimeFormat string
	NoColor    bool
}

func (w *ModuleConsoleWriter) Write(p []byte) (n int, err error) {
	var event map[string]any
	if err := json.Unmarshal(p, &event); err != nil {
		// If not JSON, write as-is
		return w.Out.Write(p)
	}

	// Extract module and remove it from event so ConsoleWriter doesn't print it
	module := ""
	if m, ok := event["module"].(string); ok {
		module = m
		delete(event, "module")
	}

	// Re-marshal the event without module
	modifiedJSON, err := json.Marshal(event)
	if err != nil {
		return w.Out.Write(p)
	}

	// Use zerolog's ConsoleWriter for default coloring
	var buf strings.Builder
	tempCW := zerolog.ConsoleWriter{
		Out:        &buf,
		TimeFormat: w.TimeFormat,
		NoColor:    w.NoColor,
		FormatCaller: func(i any) string {
			var c string
			if cc, ok := i.(string); ok {
				c = cc
			}
			if len(c) > 0 {
				if parts := strings.Split(c, "/"); len(parts) > 0 {
					c = parts[len(parts)-1]
				}
			}
			return c
		},
	}

	// Write the modified event to buffer
	tempCW.Write(modifiedJSON)
	output := buf.String()

	// Insert module after the level if present
	if module != "" {
		// Format is typically: "timestamp LEVEL message fields..."
		// We need to insert [module] after LEVEL
		parts := strings.SplitN(output, " ", 3)
		if len(parts) >= 2 {
			// Color codes for module: bold yellow/orange
			const (
				colorReset  = "\x1b[0m"
				colorBold   = "\x1b[1m"
				colorOrange = "\x1b[33m"
			)

			// Reconstruct: timestamp LEVEL [module] rest
			var result strings.Builder
			result.WriteString(parts[0])
			result.WriteString(" ")
			result.WriteString(parts[1])
			result.WriteString(" ")

			// Add colored and bold module
			if !w.NoColor {
				result.WriteString(colorBold)
				result.WriteString(colorOrange)
			}
			result.WriteString("[")
			result.WriteString(module)
			result.WriteString("]")
			if !w.NoColor {
				result.WriteString(colorReset)
			}

			if len(parts) > 2 {
				result.WriteString(" ")
				result.WriteString(parts[2])
			} else {
				result.WriteString("\n")
			}
			output = result.String()
		}
	}

	_, err = w.Out.Write([]byte(output))
	return len(p), err
}
