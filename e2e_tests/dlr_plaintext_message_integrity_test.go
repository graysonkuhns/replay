package cmd_test

import (
	"fmt"
	"strings"
	"testing"

	"replay/e2e_tests/testhelpers"
)

func TestDLRPlaintextMessageIntegrity(t *testing.T) {
	t.Parallel()
	// Test to verify that the plaintext body content of moved messages remains unchanged when using the DLR operation.
	baseTest := testhelpers.NewBaseE2ETest(t, "dlr_plaintext_integrity_test")

	// Prepare messages with unique plaintext body content using the builder.
	numMessages := 3

	builder := testhelpers.NewTestMessageBuilder().
		WithAttributes(map[string]string{"testRun": baseTest.TestRunID})

	var expectedBodies []string
	for i := 1; i <= numMessages; i++ {
		body := fmt.Sprintf("DLR Plaintext Integrity Test message %d", i)
		expectedBodies = append(expectedBodies, body)
		builder.WithTextMessage(body)
	}

	messages := builder.Build()

	if err := baseTest.PublishAndWait(messages); err != nil {
		t.Fatalf("Failed to publish test messages: %v", err)
	}

	// Simulate user inputs: "m" (move) for all messages
	inputs := strings.Repeat("m\n", numMessages)

	// Run the dlr command.
	actual, err := baseTest.RunDLRCommand(inputs)
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}

	// Verify the output structure without assuming message order
	if !strings.Contains(actual, fmt.Sprintf("Starting DLR review from %s", baseTest.Setup.GetSourceSubscriptionName())) {
		t.Fatalf("Expected DLR start message not found in output")
	}

	// Count occurrences of moved messages
	movedCount := strings.Count(actual, "moved successfully")
	if movedCount != numMessages {
		t.Fatalf("Expected %d 'moved successfully' messages, got %d", numMessages, movedCount)
	}

	// Verify all test messages appear in the output (in any order)
	for _, expectedBody := range expectedBodies {
		if !strings.Contains(actual, expectedBody) {
			t.Fatalf("Expected message '%s' not found in output", expectedBody)
		}
	}

	// Verify final summary
	expectedSummary := fmt.Sprintf("Dead-lettered messages review completed. Total messages processed: %d", numMessages)
	if !strings.Contains(actual, expectedSummary) {
		t.Fatalf("Expected summary '%s' not found", expectedSummary)
	}
	t.Logf("DLR command executed for body integrity test")

	// Allow time for moved messages to propagate.
	baseTest.WaitForMessagePropagation()

	// Poll the destination subscription for moved messages.
	received, err := baseTest.GetMessagesFromDestination(numMessages)
	if err != nil {
		t.Fatalf("Error receiving messages from destination: %v", err)
	}

	// Verify that each expected message body is found in the received messages, regardless of order.
	for _, expected := range expectedBodies {
		found := false
		for _, msg := range received {
			if string(msg.Data) == expected {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Expected message body '%s' not found in received messages", expected)
		}
	}

	// Verify that the source subscription is empty (all messages were moved)
	if err := baseTest.VerifyMessagesInSource(0); err != nil {
		t.Fatalf("%v", err)
	}

	t.Logf("Plaintext message body integrity verified for all %d messages moved using DLR operation", numMessages)
}
