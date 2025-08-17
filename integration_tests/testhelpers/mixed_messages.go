package testhelpers

import (
	"time"

	"cloud.google.com/go/pubsub"
)

// CreateMixedTestMessages demonstrates the power of the TestMessageBuilder
// by creating a variety of message types for comprehensive testing.
func CreateMixedTestMessages(testRun string) []pubsub.Message {
	builder := NewTestMessageBuilder().
		WithAttributes(map[string]string{"testRun": testRun})

	// Add different types of messages
	messages := builder.
		// Simple text messages
		WithAttribute("messageType", "greeting").
		WithTextMessage("Hello, World!").
		WithAttribute("messageType", "farewell").
		WithTextMessage("Goodbye, World!").

		// JSON messages with different structures
		WithAttribute("messageType", "user").
		WithJSONMessage(map[string]interface{}{
			"id":       12345,
			"username": "testuser",
			"email":    "test@example.com",
			"active":   true,
			"created":  time.Now().Unix(),
		}).
		WithAttribute("messageType", "config").
		WithJSONMessage(map[string]interface{}{
			"settings": map[string]interface{}{
				"theme":         "dark",
				"notifications": true,
				"features":      []string{"beta", "experimental"},
			},
			"version": "1.0.0",
		}).

		// Binary messages of different sizes
		WithAttribute("messageType", "smallBinary").
		WithBinaryMessage(64).
		WithAttribute("messageType", "largeBinary").
		WithPatternBinaryMessage(1024).
		Build()

	return messages
}
