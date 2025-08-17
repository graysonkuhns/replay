package cmd_test

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"replay/integration_tests/testhelpers"

	"cloud.google.com/go/pubsub/v2"
)

func TestDLRBinaryEdgeCases(t *testing.T) {
	t.Parallel()
	// Test specific binary edge cases that might cause issues in message processing
	setup := testhelpers.SetupIntegrationTest(t)
	testRunValue := "dlr_binary_edge_cases_test"

	// Purge any existing messages from the source subscription to ensure test isolation
	if err := testhelpers.PurgeSubscription(setup.Context, setup.Client, setup.GetSourceSubscriptionName()); err != nil {
		t.Fatalf("Failed to purge source subscription: %v", err)
	}

	// Prepare edge case binary messages
	numMessages := 4
	sourceTopicName := setup.GetSourceTopicName()
	var messages []pubsub.Message
	var expectedBinaryData [][]byte
	var descriptions []string

	// 1. Message with null bytes only
	nullBytes := make([]byte, 128)
	// All bytes are already 0 by default
	expectedBinaryData = append(expectedBinaryData, nullBytes)
	descriptions = append(descriptions, "null bytes only")

	// 2. Message with high ASCII/extended ASCII characters
	highASCII := make([]byte, 128)
	for i := 0; i < len(highASCII); i++ {
		highASCII[i] = byte(128 + (i % 128)) // Values from 128-255
	}
	expectedBinaryData = append(expectedBinaryData, highASCII)
	descriptions = append(descriptions, "high ASCII bytes")

	// 3. Message with control characters
	controlChars := make([]byte, 128)
	for i := 0; i < len(controlChars); i++ {
		controlChars[i] = byte(i % 32) // Control characters (0-31)
	}
	expectedBinaryData = append(expectedBinaryData, controlChars)
	descriptions = append(descriptions, "control characters")

	// 4. Empty message (0 bytes)
	emptyData := make([]byte, 0)
	expectedBinaryData = append(expectedBinaryData, emptyData)
	descriptions = append(descriptions, "empty message")

	// Create pubsub messages with binary payloads
	for i, binaryData := range expectedBinaryData {
		// For logging, encode the data to base64
		base64Data := base64.StdEncoding.EncodeToString(binaryData)

		messages = append(messages, pubsub.Message{
			Data: binaryData,
			Attributes: map[string]string{
				"testRun":      testRunValue,
				"contentType":  "application/octet-stream",
				"messageIndex": fmt.Sprintf("%d", i+1),
				"sizeBytes":    fmt.Sprintf("%d", len(binaryData)),
				"description":  descriptions[i],
				"base64Data":   base64Data, // Store full base64 for empty or small payloads
			},
		})
	}

	_, err := testhelpers.PublishTestMessages(setup.Context, setup.Client, sourceTopicName, messages, "binary-edge-test-key")
	if err != nil {
		t.Fatalf("Failed to publish binary test messages: %v", err)
	}
	time.Sleep(30 * time.Second) // Wait for messages to arrive in the subscription

	// Prepare CLI arguments for the dlr command.
	dlrArgs := []string{
		"dlr",
		"--source-type", "GCP_PUBSUB_SUBSCRIPTION",
		"--destination-type", "GCP_PUBSUB_TOPIC",
		"--source", setup.GetSourceSubscriptionName(),
		"--destination", setup.GetDestTopicName(),
	}

	// Simulate user inputs: "m" (move) for all messages
	var inputs string
	for i := 0; i < numMessages; i++ {
		inputs += "m\n"
	}

	simulator, err := testhelpers.NewStdinSimulator(inputs)
	if err != nil {
		t.Fatalf("Failed to create stdin simulator: %v", err)
	}
	defer simulator.Cleanup()

	// Run the dlr command.
	_, err = testhelpers.RunCLICommand(dlrArgs)
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}

	t.Logf("DLR command executed for binary edge cases test")

	// Allow time for moved messages to propagate.
	time.Sleep(20 * time.Second)

	// Poll the destination subscription for moved messages.
	received, err := testhelpers.PollMessages(setup.Context, setup.Client, setup.GetDestSubscriptionName(), testRunValue, numMessages)
	if err != nil {
		t.Fatalf("Error receiving messages from destination: %v", err)
	}
	if len(received) != numMessages {
		t.Fatalf("Expected %d messages in destination, got %d", numMessages, len(received))
	}

	// Verify that each binary edge case maintains its data integrity
	for i, expectedData := range expectedBinaryData {
		found := false
		desc := descriptions[i]

		for _, msg := range received {
			if msg.Attributes["description"] == desc && bytes.Equal(msg.Data, expectedData) {
				found = true
				t.Logf("Binary edge case '%s' verified (size: %d bytes)",
					desc, len(expectedData))
				break
			}
		}

		if !found {
			// Convert to base64 for error messaging since these are odd byte sequences
			base64Data := base64.StdEncoding.EncodeToString(expectedData)
			t.Fatalf("Binary edge case '%s' failed integrity check (base64: %s)",
				desc, base64Data)
		}
	}

	// Verify that the source subscription is empty (all messages were moved)
	sourceReceived, err := testhelpers.PollMessages(setup.Context, setup.Client, setup.GetSourceSubscriptionName(), testRunValue, 0)
	if err != nil {
		t.Fatalf("Error polling source subscription: %v", err)
	}
	if len(sourceReceived) != 0 {
		t.Fatalf("Expected 0 messages in source subscription, got %d", len(sourceReceived))
	}

	t.Logf("Binary edge cases verified for all %d messages moved using DLR operation", numMessages)
}
