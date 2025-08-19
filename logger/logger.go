package logger

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

// Logger defines the interface for structured logging
type Logger interface {
	// Info logs an informational message
	Info(msg string, fields ...Field)

	// Error logs an error message
	Error(msg string, err error, fields ...Field)

	// Debug logs a debug message
	Debug(msg string, fields ...Field)

	// WithContext returns a logger with context fields
	WithContext(ctx context.Context) Logger

	// WithFields returns a logger with additional fields
	WithFields(fields ...Field) Logger
}

// Field represents a key-value pair for structured logging
type Field struct {
	Key   string
	Value interface{}
}

// String creates a string field
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int creates an integer field
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Error creates an error field
func Error(err error) Field {
	return Field{Key: "error", Value: err}
}

// Any creates a field with any value
func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// defaultLogger implements the Logger interface using the standard log package
type defaultLogger struct {
	logger *log.Logger
	fields []Field
}

// NewLogger creates a new logger instance
func NewLogger() Logger {
	return &defaultLogger{
		logger: log.New(os.Stdout, "", log.LstdFlags),
		fields: []Field{},
	}
}

// NewLoggerWithOutput creates a new logger instance with custom output
func NewLoggerWithOutput(w io.Writer) Logger {
	return &defaultLogger{
		logger: log.New(w, "", log.LstdFlags),
		fields: []Field{},
	}
}

// Info logs an informational message
func (l *defaultLogger) Info(msg string, fields ...Field) {
	l.log("INFO", msg, fields...)
}

// Error logs an error message
func (l *defaultLogger) Error(msg string, err error, fields ...Field) {
	allFields := append([]Field{Error(err)}, fields...)
	l.log("ERROR", msg, allFields...)
}

// Debug logs a debug message
func (l *defaultLogger) Debug(msg string, fields ...Field) {
	l.log("DEBUG", msg, fields...)
}

// WithContext returns a logger with context fields
func (l *defaultLogger) WithContext(ctx context.Context) Logger {
	// In a real implementation, you might extract request ID, trace ID, etc. from context
	return l
}

// WithFields returns a logger with additional fields
func (l *defaultLogger) WithFields(fields ...Field) Logger {
	newFields := make([]Field, len(l.fields)+len(fields))
	copy(newFields, l.fields)
	copy(newFields[len(l.fields):], fields)

	return &defaultLogger{
		logger: l.logger,
		fields: newFields,
	}
}

// log is the internal logging method
func (l *defaultLogger) log(level, msg string, fields ...Field) {
	// Combine persistent fields with one-time fields
	allFields := append(l.fields, fields...)

	// Format the log entry
	timestamp := time.Now().Format(time.RFC3339)
	logEntry := fmt.Sprintf("[%s] %s: %s", timestamp, level, msg)

	// Add fields
	if len(allFields) > 0 {
		logEntry += " {"
		for i, field := range allFields {
			if i > 0 {
				logEntry += ", "
			}
			logEntry += fmt.Sprintf("%s=%v", field.Key, field.Value)
		}
		logEntry += "}"
	}

	// Use the underlying logger without timestamp (we added our own)
	l.logger.SetFlags(0)
	l.logger.Println(logEntry)
	l.logger.SetFlags(log.LstdFlags)
}
