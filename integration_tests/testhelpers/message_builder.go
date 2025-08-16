package testhelpers

import (
	"crypto/rand"
	"encoding/json"
	"fmt"

	"cloud.google.com/go/pubsub"
)

// TestMessageBuilder provides a fluent interface for building test messages.
type TestMessageBuilder struct {
	messages   []pubsub.Message
	attributes map[string]string
}

// NewTestMessageBuilder creates a new TestMessageBuilder instance.
func NewTestMessageBuilder() *TestMessageBuilder {
	return &TestMessageBuilder{
		messages:   make([]pubsub.Message, 0),
		attributes: make(map[string]string),
	}
}

// WithTextMessage adds a text message to the builder.
func (b *TestMessageBuilder) WithTextMessage(content string) *TestMessageBuilder {
	message := pubsub.Message{
		Data:       []byte(content),
		Attributes: b.copyAttributes(),
	}
	b.messages = append(b.messages, message)
	return b
}

// WithJSONMessage adds a JSON message to the builder by marshaling the provided data.
func (b *TestMessageBuilder) WithJSONMessage(data interface{}) *TestMessageBuilder {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		// In test context, we'll panic on marshal errors since it indicates test setup issues
		panic(fmt.Sprintf("Failed to marshal JSON data: %v", err))
	}

	attrs := b.copyAttributes()
	if attrs["contentType"] == "" {
		attrs["contentType"] = "application/json"
	}

	message := pubsub.Message{
		Data:       jsonBytes,
		Attributes: attrs,
	}
	b.messages = append(b.messages, message)
	return b
}

// WithBinaryMessage adds a binary message of the specified size to the builder.
// The binary data is randomly generated.
func (b *TestMessageBuilder) WithBinaryMessage(size int) *TestMessageBuilder {
	binaryData := make([]byte, size)
	if _, err := rand.Read(binaryData); err != nil {
		panic(fmt.Sprintf("Failed to generate binary data: %v", err))
	}

	attrs := b.copyAttributes()
	if attrs["contentType"] == "" {
		attrs["contentType"] = "application/octet-stream"
	}
	if attrs["sizeBytes"] == "" {
		attrs["sizeBytes"] = fmt.Sprintf("%d", size)
	}

	message := pubsub.Message{
		Data:       binaryData,
		Attributes: attrs,
	}
	b.messages = append(b.messages, message)
	return b
}

// WithPatternBinaryMessage adds a binary message with a deterministic byte pattern.
// This is useful for tests that need predictable binary content.
func (b *TestMessageBuilder) WithPatternBinaryMessage(size int) *TestMessageBuilder {
	binaryData := make([]byte, size)
	for i := 0; i < size; i++ {
		binaryData[i] = byte(i % 256)
	}

	attrs := b.copyAttributes()
	if attrs["contentType"] == "" {
		attrs["contentType"] = "application/octet-stream"
	}
	if attrs["sizeBytes"] == "" {
		attrs["sizeBytes"] = fmt.Sprintf("%d", size)
	}

	message := pubsub.Message{
		Data:       binaryData,
		Attributes: attrs,
	}
	b.messages = append(b.messages, message)
	return b
}

// WithAttributes sets attributes that will be applied to subsequently added messages.
// This returns the builder for chaining.
func (b *TestMessageBuilder) WithAttributes(attrs map[string]string) *TestMessageBuilder {
	for k, v := range attrs {
		b.attributes[k] = v
	}
	return b
}

// WithAttribute sets a single attribute that will be applied to subsequently added messages.
func (b *TestMessageBuilder) WithAttribute(key, value string) *TestMessageBuilder {
	b.attributes[key] = value
	return b
}

// Build returns the slice of messages that have been built.
func (b *TestMessageBuilder) Build() []pubsub.Message {
	// Return a copy to prevent modification of the builder's internal state
	result := make([]pubsub.Message, len(b.messages))
	copy(result, b.messages)
	return result
}

// Count returns the number of messages in the builder.
func (b *TestMessageBuilder) Count() int {
	return len(b.messages)
}

// Reset clears all messages and attributes from the builder.
func (b *TestMessageBuilder) Reset() *TestMessageBuilder {
	b.messages = make([]pubsub.Message, 0)
	b.attributes = make(map[string]string)
	return b
}

// copyAttributes creates a copy of the current attributes map.
func (b *TestMessageBuilder) copyAttributes() map[string]string {
	attrs := make(map[string]string)
	for k, v := range b.attributes {
		attrs[k] = v
	}
	return attrs
}
