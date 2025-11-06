// Package logger provides a simple structured logger
package logger

import (
	"fmt"
	"log"
	"os"
)

// Logger provides structured logging
type Logger struct {
	*log.Logger
	enabled bool
}

// New creates a new Logger instance
func New() *Logger {
	return &Logger{
		Logger:  log.New(os.Stdout, "[EVE-O-Provit] ", log.LstdFlags),
		enabled: true,
	}
}

// NewNoop creates a no-op logger for testing
func NewNoop() *Logger {
	return &Logger{
		Logger:  log.New(os.Stdout, "", 0),
		enabled: false,
	}
}

// Debug logs debug-level messages with key-value pairs
func (l *Logger) Debug(msg string, keysAndValues ...interface{}) {
	if !l.enabled {
		return
	}
	l.logWithKV("DEBUG", msg, keysAndValues...)
}

// Info logs info-level messages with key-value pairs
func (l *Logger) Info(msg string, keysAndValues ...interface{}) {
	if !l.enabled {
		return
	}
	l.logWithKV("INFO", msg, keysAndValues...)
}

// Warn logs warning-level messages with key-value pairs
func (l *Logger) Warn(msg string, keysAndValues ...interface{}) {
	if !l.enabled {
		return
	}
	l.logWithKV("WARN", msg, keysAndValues...)
}

// Error logs error-level messages with key-value pairs
func (l *Logger) Error(msg string, keysAndValues ...interface{}) {
	if !l.enabled {
		return
	}
	l.logWithKV("ERROR", msg, keysAndValues...)
}

// logWithKV formats and logs messages with key-value pairs
func (l *Logger) logWithKV(level, msg string, keysAndValues ...interface{}) {
	output := level + " " + msg

	// Add key-value pairs
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			output += " " + keysAndValues[i].(string) + "=" + formatValue(keysAndValues[i+1])
		}
	}

	l.Println(output)
}

// formatValue formats a value for logging
func formatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case int, int32, int64:
		return fmt.Sprint(val)
	case float32, float64:
		return fmt.Sprint(val)
	case error:
		return val.Error()
	default:
		return fmt.Sprint(val)
	}
}
