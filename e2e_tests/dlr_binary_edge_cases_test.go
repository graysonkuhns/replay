package cmd_test

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"replay/e2e_tests/testhelpers"

	"cloud.google.com/go/pubsub/v2"
)

func TestDLRBinaryEdgeCases(t *testing.T) {
	t.Parallel()
	// Test specific binary edge cases that might cause issues in message processing
	baseTest := testhelpers.NewBaseE2ETest(t, "dlr_binary_edge_cases_test")

	// Purge any existing messages from the source subscription to ensure test isolation
	if err := testhelpers.PurgeSubscription(baseTest.Setup.Context, baseTest.Setup.Client, baseTest.Setup.GetSourceSubscriptionName()); err != nil {
		t.Fatalf("Failed to purge source subscription: %v", err)
	}

	// Prepare edge case binary messages
	numMessages := 4
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
				"testRun":      baseTest.TestRunID,
				"contentType":  "application/octet-stream",
				"messageIndex": fmt.Sprintf("%d", i+1),
				"sizeBytes":    fmt.Sprintf("%d", len(binaryData)),
				"description":  descriptions[i],
				"base64Data":   base64Data, // Store full base64 for empty or small payloads
			},
		})
	}

	if err := baseTest.PublishAndWait(messages); err != nil {
		t.Fatalf("Failed to publish binary test messages: %v", err)
	}

	// Simulate user inputs: "m" (move) for all messages
	inputs := strings.Repeat("m\n", numMessages)

	// Run the dlr command.
	_, err := baseTest.RunDLRCommand(inputs)
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}

	t.Logf("DLR command executed for binary edge cases test")

	// Allow time for moved messages to propagate.
	baseTest.WaitForMessagePropagation()
	// Add extra wait for edge cases - binary messages may need more time
	baseTest.WaitForMessagePropagation()
	// Additional wait specifically for binary edge cases
	baseTest.WaitForMessagePropagation()

	// Poll the destination subscription for moved messages.
	received, err := baseTest.GetMessagesFromDestination(numMessages)
	if err != nil {
		t.Logf("Error receiving messages from destination: %v", err)
		// Log which messages were received for debugging
		partialReceived, partialErr := baseTest.GetMessagesFromDestination(0)
		if partialErr != nil {
			t.Logf("Could not retrieve any messages for debugging: %v", partialErr)
		} else {
			t.Logf("Received %d messages for debugging:", len(partialReceived))
			for i, msg := range partialReceived {
				t.Logf("  Message %d - Description: %s, Size: %d bytes, TestRun: %s",
					i+1, msg.Attributes["description"], len(msg.Data), msg.Attributes["testRun"])
			}
		}
		t.Fatalf("Error receiving messages from destination: %v", err)
	}

	// Verify that each binary edge case maintains its data integrity
	for i, expectedData := range expectedBinaryData {
		found := false
		desc := descriptions[i]

		for _, msg := range received {
			if msg.Attributes["description"] == desc && bytes.Equal(msg.Data, expectedData) {
				found = true
				// Use AssertBinaryEquals for better error reporting if needed
				testhelpers.AssertBinaryEquals(t, msg.Data, expectedData)
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
	if err := baseTest.VerifyMessagesInSource(0); err != nil {
		t.Fatalf("%v", err)
	}

	t.Logf("Binary edge cases verified for all %d messages moved using DLR operation", numMessages)
}
