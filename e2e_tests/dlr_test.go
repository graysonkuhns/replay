package cmd_test

import (
	"fmt"
	"strings"
	"testing"

	"replay/e2e_tests/testhelpers"
)

func TestDLROperation(t *testing.T) {
	t.Parallel()
	// Set up e2e test environment
	baseTest := testhelpers.NewBaseE2ETest(t, "dlr_test")

	// Publish two test messages to the dead-letter topic: one to move and one to discard.
	messages := baseTest.CreateTestMessages(2, "Test message")
	messages[0].Data = []byte("Test message move")
	messages[1].Data = []byte("Test message discard")

	if err := baseTest.PublishAndWait(messages); err != nil {
		t.Fatalf("Failed to publish test messages: %v", err)
	}

	// Simulate user inputs: "m" for moving message 1 and "d" for discarding message 2.
	actual, err := baseTest.RunDLRCommand("m\nd\n")
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}

	// Verify the output structure without assuming message order
	if !strings.Contains(actual, fmt.Sprintf("Starting DLR review from %s", baseTest.Setup.GetSourceSubscriptionName())) {
		t.Fatalf("Expected DLR start message not found in output")
	}

	// Count occurrences of each action result
	movedCount := strings.Count(actual, "moved successfully")
	discardedCount := strings.Count(actual, "discarded (acked)")

	if movedCount != 1 {
		t.Fatalf("Expected 1 'moved successfully' message, got %d", movedCount)
	}
	if discardedCount != 1 {
		t.Fatalf("Expected 1 'discarded (acked)' message, got %d", discardedCount)
	}

	// Verify both test messages appear in the output (in any order)
	if !strings.Contains(actual, "Test message move") {
		t.Fatalf("Expected 'Test message move' not found in output")
	}
	if !strings.Contains(actual, "Test message discard") {
		t.Fatalf("Expected 'Test message discard' not found in output")
	}

	// Verify final summary
	if !strings.Contains(actual, "Dead-lettered messages review completed. Total messages processed: 2") {
		t.Fatalf("Expected summary with 2 processed messages not found")
	}

	// Allow time for the moved message to propagate.
	baseTest.WaitForMessagePropagation()

	// Verify the message was published to the destination topic via its subscription.
	if err := baseTest.VerifyMessagesInDestination(1); err != nil {
		t.Fatalf("%v", err)
	}

	// Verify that the discarded message is no longer in the source subscription.
	if err := baseTest.VerifyMessagesInSource(0); err != nil {
		t.Fatalf("%v", err)
	}
}
