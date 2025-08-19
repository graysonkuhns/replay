package cmd_test

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"replay/e2e_tests/testhelpers"
)

func TestDLRBinaryMessageIntegrity(t *testing.T) {
	t.Parallel()
	// Test to verify that binary data remains unchanged when moved using the DLR operation.
	baseTest := testhelpers.NewBaseE2ETest(t, "dlr_binary_integrity_test")

	// Prepare various binary message payloads using the builder.
	numMessages := 2

	builder := testhelpers.NewTestMessageBuilder().
		WithAttributes(map[string]string{"testRun": baseTest.TestRunID})

	var expectedBinaryData [][]byte

	// 1. Small binary data (16 bytes) - random
	builder.WithAttribute("messageIndex", "1").
		WithBinaryMessage(16)
	// Since we can't predict random data, we'll get it from the built messages

	// 2. Medium binary data with specific pattern (1KB) - deterministic
	builder.WithAttribute("messageIndex", "2").
		WithPatternBinaryMessage(1024)

	messages := builder.Build()

	// Extract expected binary data from the built messages
	for _, msg := range messages {
		expectedBinaryData = append(expectedBinaryData, msg.Data)
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

	t.Logf("DLR command executed for binary data integrity test")

	// Allow time for moved messages to propagate.
	baseTest.WaitForMessagePropagation()

	// Poll the destination subscription for moved messages.
	received, err := baseTest.GetMessagesFromDestination(numMessages)
	if err != nil {
		t.Fatalf("Error receiving messages from destination: %v", err)
	}

	// Verify that each binary message maintains its exact data integrity
	// Match messages by their size and/or sample attributes
	for i, expectedData := range expectedBinaryData {
		found := false
		expectedSize := len(expectedData)

		for _, msg := range received {
			// Use size as first filter since binary comparison can be expensive
			if msg.Attributes["sizeBytes"] == fmt.Sprintf("%d", expectedSize) {
				// If sizes match, do full binary comparison
				if bytes.Equal(msg.Data, expectedData) {
					found = true
					// Now use AssertBinaryEquals to provide better error message if needed
					testhelpers.AssertBinaryEquals(t, msg.Data, expectedData)
					t.Logf("Binary message %d verified (size: %d bytes)", i+1, expectedSize)
					break
				} else {
					// If sizes match but content doesn't, log more details
					t.Logf("Binary message size matches (%d bytes) but content differs", expectedSize)
				}
			}
		}

		if !found {
			expectedSample := "unknown"
			if len(expectedData) > 0 {
				sampleSize := 32
				if len(expectedData) < sampleSize {
					sampleSize = len(expectedData)
				}
				expectedSample = base64.StdEncoding.EncodeToString(expectedData[:sampleSize])
			}

			t.Fatalf("Binary message %d (size: %d bytes, sample: %s) not found or corrupted",
				i+1, expectedSize, expectedSample)
		}
	}

	// Verify that the source subscription is empty (all messages were moved)
	if err := baseTest.VerifyMessagesInSource(0); err != nil {
		t.Fatalf("%v", err)
	}

	t.Logf("Binary message integrity verified for all %d messages moved using DLR operation", numMessages)
}
