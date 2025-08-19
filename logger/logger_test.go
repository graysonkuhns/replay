package logger

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestLogger(t *testing.T) {
	// Create a buffer to capture output
	buf := &bytes.Buffer{}
	log := NewLoggerWithOutput(buf)

	// Test Info logging
	log.Info("Test info message", String("key", "value"), Int("count", 42))
	output := buf.String()
	if !strings.Contains(output, "INFO") {
		t.Errorf("Expected INFO in output, got: %s", output)
	}
	if !strings.Contains(output, "Test info message") {
		t.Errorf("Expected message in output, got: %s", output)
	}
	if !strings.Contains(output, "key=value") {
		t.Errorf("Expected key=value in output, got: %s", output)
	}
	if !strings.Contains(output, "count=42") {
		t.Errorf("Expected count=42 in output, got: %s", output)
	}

	// Clear buffer
	buf.Reset()

	// Test Error logging
	testErr := errors.New("test error")
	log.Error("Test error message", testErr, String("operation", "test"))
	output = buf.String()
	if !strings.Contains(output, "ERROR") {
		t.Errorf("Expected ERROR in output, got: %s", output)
	}
	if !strings.Contains(output, "error=test error") {
		t.Errorf("Expected error=test error in output, got: %s", output)
	}
	if !strings.Contains(output, "operation=test") {
		t.Errorf("Expected operation=test in output, got: %s", output)
	}

	// Clear buffer
	buf.Reset()

	// Test WithFields
	logWithFields := log.WithFields(String("persistent", "field"))
	logWithFields.Info("Message with persistent field", String("temporary", "field"))
	output = buf.String()
	if !strings.Contains(output, "persistent=field") {
		t.Errorf("Expected persistent=field in output, got: %s", output)
	}
	if !strings.Contains(output, "temporary=field") {
		t.Errorf("Expected temporary=field in output, got: %s", output)
	}
}
