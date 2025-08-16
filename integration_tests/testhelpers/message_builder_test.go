package testhelpers_test

import (
	"encoding/json"
	"testing"

	"replay/integration_tests/testhelpers"
)

func TestTestMessageBuilder(t *testing.T) {
	// Test basic builder functionality
	builder := testhelpers.NewTestMessageBuilder()

	// Test count starts at 0
	if builder.Count() != 0 {
		t.Fatalf("Expected count 0, got %d", builder.Count())
	}

	// Test WithTextMessage
	builder.WithTextMessage("Hello, World!")
	if builder.Count() != 1 {
		t.Fatalf("Expected count 1 after adding text message, got %d", builder.Count())
	}

	// Test WithAttributes and chaining
	builder.WithAttributes(map[string]string{
		"testRun": "test-123",
		"priority": "high",
	}).WithTextMessage("Second message")

	if builder.Count() != 2 {
		t.Fatalf("Expected count 2 after chaining, got %d", builder.Count())
	}

	// Test Build() returns the correct messages
	messages := builder.Build()
	if len(messages) != 2 {
		t.Fatalf("Expected 2 messages from Build(), got %d", len(messages))
	}

	// Verify first message
	if string(messages[0].Data) != "Hello, World!" {
		t.Fatalf("Expected first message 'Hello, World!', got '%s'", string(messages[0].Data))
	}
	if messages[0].Attributes["testRun"] != "" {
		// First message should not have attributes since they were added after
		t.Logf("First message attributes: %v", messages[0].Attributes)
	}

	// Verify second message has attributes
	if string(messages[1].Data) != "Second message" {
		t.Fatalf("Expected second message 'Second message', got '%s'", string(messages[1].Data))
	}
	if messages[1].Attributes["testRun"] != "test-123" {
		t.Fatalf("Expected testRun 'test-123', got '%s'", messages[1].Attributes["testRun"])
	}
	if messages[1].Attributes["priority"] != "high" {
		t.Fatalf("Expected priority 'high', got '%s'", messages[1].Attributes["priority"])
	}
}

func TestTestMessageBuilderJSON(t *testing.T) {
	builder := testhelpers.NewTestMessageBuilder()

	testData := map[string]interface{}{
		"id": 123,
		"name": "Test JSON",
		"active": true,
	}

	builder.WithJSONMessage(testData)
	messages := builder.Build()

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	// Verify JSON was marshaled correctly
	var parsed map[string]interface{}
	if err := json.Unmarshal(messages[0].Data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if parsed["id"].(float64) != 123 {
		t.Fatalf("Expected id 123, got %v", parsed["id"])
	}
	if parsed["name"].(string) != "Test JSON" {
		t.Fatalf("Expected name 'Test JSON', got %v", parsed["name"])
	}
	if parsed["active"].(bool) != true {
		t.Fatalf("Expected active true, got %v", parsed["active"])
	}

	// Verify content type was set
	if messages[0].Attributes["contentType"] != "application/json" {
		t.Fatalf("Expected contentType 'application/json', got '%s'", messages[0].Attributes["contentType"])
	}
}

func TestTestMessageBuilderBinary(t *testing.T) {
	builder := testhelpers.NewTestMessageBuilder()

	// Test random binary message
	builder.WithBinaryMessage(100)
	messages := builder.Build()

	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}

	if len(messages[0].Data) != 100 {
		t.Fatalf("Expected binary data length 100, got %d", len(messages[0].Data))
	}

	if messages[0].Attributes["contentType"] != "application/octet-stream" {
		t.Fatalf("Expected contentType 'application/octet-stream', got '%s'", messages[0].Attributes["contentType"])
	}

	if messages[0].Attributes["sizeBytes"] != "100" {
		t.Fatalf("Expected sizeBytes '100', got '%s'", messages[0].Attributes["sizeBytes"])
	}

	// Test pattern binary message
	builder.Reset().WithPatternBinaryMessage(256)
	patternMessages := builder.Build()

	if len(patternMessages) != 1 {
		t.Fatalf("Expected 1 pattern message, got %d", len(patternMessages))
	}

	if len(patternMessages[0].Data) != 256 {
		t.Fatalf("Expected pattern binary data length 256, got %d", len(patternMessages[0].Data))
	}

	// Verify pattern (first few bytes should be 0, 1, 2, 3...)
	for i := 0; i < 10; i++ {
		if patternMessages[0].Data[i] != byte(i) {
			t.Fatalf("Expected byte %d at position %d, got %d", i, i, patternMessages[0].Data[i])
		}
	}
}

func TestTestMessageBuilderReset(t *testing.T) {
	builder := testhelpers.NewTestMessageBuilder()

	builder.WithTextMessage("Message 1")
	builder.WithAttributes(map[string]string{"key": "value"})
	builder.WithTextMessage("Message 2")

	if builder.Count() != 2 {
		t.Fatalf("Expected count 2 before reset, got %d", builder.Count())
	}

	builder.Reset()

	if builder.Count() != 0 {
		t.Fatalf("Expected count 0 after reset, got %d", builder.Count())
	}

	// Verify attributes were also reset
	builder.WithTextMessage("New Message")
	messages := builder.Build()

	if len(messages[0].Attributes) != 0 {
		t.Fatalf("Expected no attributes after reset, got %v", messages[0].Attributes)
	}
}