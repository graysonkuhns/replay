package cmd_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"testing"
	"time"

	"replay/integration_tests/testhelpers"

	"cloud.google.com/go/pubsub"
)

func TestMoveBinaryMessageIntegrity(t *testing.T) {
	// Test to verify that binary data remains unchanged when moving messages using the move operation.
	ctx := context.Background()
	projectID := os.Getenv("GCP_PROJECT")
	if projectID == "" {
		t.Fatal("GCP_PROJECT environment variable must be set")
	}

	// Define test parameters.
	sourceTopicName := "default-events-dead-letter"
	sourceSubName := "default-events-dead-letter-subscription"
	destTopicName := "default-events"
	destSubName := "default-events-subscription"
	testRunValue := "move_binary_integrity_test"

	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		t.Fatalf("Failed to create PubSub client: %v", err)
	}
	defer client.Close()

	// Purge subscriptions.
	sourceSub := client.Subscription(sourceSubName)
	if err := testhelpers.PurgeSubscription(ctx, sourceSub); err != nil {
		t.Fatalf("Failed to purge source subscription: %v", err)
	}
	destSub := client.Subscription(destSubName)
	destCtx, destCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer destCancel()
	if err := testhelpers.PurgeSubscription(destCtx, destSub); err != nil {
		t.Fatalf("Failed to purge destination subscription: %v", err)
	}

	// Prepare different types of binary data
	numMessages := 3
	sourceTopic := client.Topic(sourceTopicName)
	var messages []pubsub.Message
	var expectedBinaryData [][]byte

	// 1. Binary data representing an image header (simulate part of a PNG file)
	// PNG signature (8 bytes) followed by IHDR chunk
	pngHeader := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, // IHDR chunk length (13 bytes)
		0x49, 0x48, 0x44, 0x52, // IHDR chunk type
		0x00, 0x00, 0x01, 0x00, // Width: 256
		0x00, 0x00, 0x01, 0x00, // Height: 256
		0x08, // Bit depth: 8
		0x06, // Color type: RGBA
		0x00, // Compression method: deflate
		0x00, // Filter method: standard
		0x00, // Interlace method: none
	}
	expectedBinaryData = append(expectedBinaryData, pngHeader)

	// 2. Zero-padded binary data (sometimes problematic for string handling)
	zeroPadded := make([]byte, 256)
	for i := 0; i < len(zeroPadded); i++ {
		// Create a pattern with some null bytes
		if i%4 == 0 {
			zeroPadded[i] = 0x00 // null byte every 4 bytes
		} else {
			zeroPadded[i] = byte(i % 256)
		}
	}
	expectedBinaryData = append(expectedBinaryData, zeroPadded)

	// 3. Random binary data with full byte range (0x00-0xFF)
	fullRange := make([]byte, 4096)
	if _, err := rand.Read(fullRange); err != nil {
		t.Fatalf("Failed to generate random binary data: %v", err)
	}
	expectedBinaryData = append(expectedBinaryData, fullRange)

	// Create pubsub messages with binary payloads
	for i, binaryData := range expectedBinaryData {
		// Include a hash and size for verification
		messages = append(messages, pubsub.Message{
			Data: binaryData,
			Attributes: map[string]string{
				"testRun":      testRunValue,
				"contentType":  "application/octet-stream",
				"messageIndex": fmt.Sprintf("%d", i+1),
				"sizeBytes":    fmt.Sprintf("%d", len(binaryData)),
				"dataSample":   base64.StdEncoding.EncodeToString(binaryData[:min(32, len(binaryData))]),
			},
		})
	}

	_, err = testhelpers.PublishTestMessages(ctx, sourceTopic, messages, "binary-move-test-key")
	if err != nil {
		t.Fatalf("Failed to publish binary test messages: %v", err)
	}
	time.Sleep(15 * time.Second) // Wait for messages to arrive in the subscription

	// Run the move command.
	moveArgs := []string{
		"move",
		"--source-type", "GCP_PUBSUB_SUBSCRIPTION",
		"--destination-type", "GCP_PUBSUB_TOPIC",
		"--source", fmt.Sprintf("projects/%s/subscriptions/%s", projectID, sourceSubName),
		"--destination", fmt.Sprintf("projects/%s/topics/%s", projectID, destTopicName),
		"--count", fmt.Sprintf("%d", numMessages),
	}

	actual, err := testhelpers.RunCLICommand(moveArgs)
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}
	t.Logf("Move command executed for binary integrity test: %s", actual)

	// Allow time for moved messages to propagate.
	time.Sleep(5 * time.Second)

	// Poll the destination subscription for moved messages.
	received, err := testhelpers.PollMessages(ctx, destSub, testRunValue, numMessages)
	if err != nil {
		t.Fatalf("Error receiving messages from destination: %v", err)
	}
	if len(received) != numMessages {
		t.Fatalf("Expected %d messages in destination, got %d", numMessages, len(received))
	}

	// Verify that each binary message maintains its data integrity
	for i, expectedData := range expectedBinaryData {
		found := false
		for _, msg := range received {
			if bytes.Equal(msg.Data, expectedData) {
				found = true
				break
			}
		}
		if !found {
			sampleBase64 := base64.StdEncoding.EncodeToString(expectedData[:min(32, len(expectedData))])
			t.Fatalf("Binary message %d (sample: %s) not found or corrupted", i+1, sampleBase64)
		}
	}

	// Verify that the source subscription is empty
	sourceReceived, err := testhelpers.PollMessages(ctx, sourceSub, testRunValue, 0)
	if err != nil {
		t.Fatalf("Error polling source subscription: %v", err)
	}
	if len(sourceReceived) != 0 {
		t.Fatalf("Expected 0 messages in source subscription, got %d", len(sourceReceived))
	}

	t.Logf("Binary message integrity verified for all %d messages moved using move operation", numMessages)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
