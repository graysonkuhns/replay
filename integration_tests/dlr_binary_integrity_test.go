package cmd_test

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"replay/integration_tests/testhelpers"
)

func TestDLRBinaryMessageIntegrity(t *testing.T) {
	t.Parallel()
	// Test to verify that binary data remains unchanged when moved using the DLR operation.
	setup := testhelpers.SetupIntegrationTest(t)
	testRunValue := "dlr_binary_integrity_test"

	// Prepare various binary message payloads using the builder.
	numMessages := 2
	sourceTopicName := setup.GetSourceTopicName()

	builder := testhelpers.NewTestMessageBuilder().
		WithAttributes(map[string]string{"testRun": testRunValue})

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

	_, err := testhelpers.PublishTestMessages(setup.Context, setup.Client, sourceTopicName, messages, "binary-test-ordering-key")
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
	origStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe for stdin: %v", err)
	}

	// Write "m" for each message to move them all
	var inputs string
	for i := 0; i < numMessages; i++ {
		inputs += "m\n"
	}

	_, err = io.WriteString(w, inputs)
	if err != nil {
		t.Fatalf("Failed to write simulated input: %v", err)
	}
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()

	// Run the dlr command.
	_, err = testhelpers.RunCLICommand(dlrArgs)
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}

	t.Logf("DLR command executed for binary data integrity test")

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
	sourceReceived, err := testhelpers.PollMessages(setup.Context, setup.Client, setup.GetSourceSubscriptionName(), testRunValue, 0)
	if err != nil {
		t.Fatalf("Error polling source subscription: %v", err)
	}
	if len(sourceReceived) != 0 {
		t.Fatalf("Expected 0 messages in source subscription, got %d", len(sourceReceived))
	}

	t.Logf("Binary message integrity verified for all %d messages moved using DLR operation", numMessages)
}
