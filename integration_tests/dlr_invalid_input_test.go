package cmd_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"replay/integration_tests/testhelpers"

	"cloud.google.com/go/pubsub/v2"
)

func TestDLRInvalidInputHandling(t *testing.T) {
	t.Parallel()
	// Test to verify that the DLR operation correctly handles invalid input by asking again
	baseTest := testhelpers.NewBaseIntegrationTest(t, "dlr_invalid_input_test")

	// Prepare 2 messages with unique body content.
	numMessages := 2
	messages := baseTest.CreateTestMessages(numMessages, "DLR Invalid Input Test message")

	if err := baseTest.PublishAndWait(messages); err != nil {
		t.Fatalf("Failed to publish test messages: %v", err)
	}

	// Simulate user inputs:
	// For message 1: invalid input "x", then "m" (move)
	// For message 2: invalid inputs "invalid", "123", then "d" (discard)
	inputs := "x\nm\ninvalid\n123\nd\n"

	// Run the dlr command.
	actual, err := baseTest.RunDLRCommand(inputs)
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}

	// Verify key behaviors in the output regardless of message order
	expectedSubstrings := []string{
		fmt.Sprintf("Starting DLR review from %s", baseTest.Setup.GetSourceSubscriptionName()),
		"Message 1:",
		"Message 2:",
		"DLR Invalid Input Test message 1",
		"DLR Invalid Input Test message 2",
		"Attributes: map[testRun:dlr_invalid_input_test]",
		"Invalid input. Please enter 'm', 'd', or 'q'.", // Should appear 3 times total
		"moved successfully",
		"discarded (acked)",
		"Dead-lettered messages review completed. Total messages processed: 2",
	}

	// Check that all expected substrings are present in the output
	for _, expected := range expectedSubstrings {
		if !strings.Contains(actual, expected) {
			t.Errorf("Expected output to contain: %s", expected)
		}
	}

	// Verify that we have exactly 3 invalid input messages (1 for first message, 2 for second)
	invalidInputCount := strings.Count(actual, "Invalid input. Please enter 'm', 'd', or 'q'.")
	if invalidInputCount != 3 {
		t.Errorf("Expected 3 'Invalid input' messages, but found %d", invalidInputCount)
	}

	// Verify that we have exactly 1 moved and 1 discarded message
	if strings.Count(actual, "moved successfully") != 1 {
		t.Errorf("Expected exactly 1 'moved successfully' message")
	}
	if strings.Count(actual, "discarded (acked)") != 1 {
		t.Errorf("Expected exactly 1 'discarded (acked)' message")
	}
	t.Logf("DLR command executed for invalid input handling test")

	// Allow time for moved messages to propagate.
	baseTest.WaitForMessagePropagation()

	// Poll the destination subscription for moved messages.
	// We expect exactly 1 message to be moved.
	received, err := baseTest.GetMessagesFromDestination(1)
	if err != nil {
		t.Fatalf("Error receiving messages from destination: %v", err)
	}

	// Verify the moved message is one of our test messages
	movedMessageData := string(received[0].Data)
	if movedMessageData != "DLR Invalid Input Test message 1" && movedMessageData != "DLR Invalid Input Test message 2" {
		t.Fatalf("Expected moved message to be one of the test messages, but got '%s'", movedMessageData)
	}

	// Verify that no messages remain in the source subscription by using a custom checking approach
	// instead of using PollMessages which expects a specific number of messages
	baseTest.WaitForMessagePropagation()

	// Create a custom receiver function to check for any messages
	var foundMessage bool
	cctx, cancel := context.WithTimeout(baseTest.Setup.Context, 10*time.Second)
	defer cancel()

	// Use Subscriber directly instead of PollMessages
	subscriber := baseTest.Setup.Client.Subscriber(baseTest.Setup.GetSourceSubscriptionName())
	err = subscriber.Receive(cctx, func(ctx context.Context, m *pubsub.Message) {
		if m.Attributes["testRun"] == baseTest.TestRunID {
			foundMessage = true
			m.Ack()
			cancel() // Stop receiving as soon as we find a matching message
		} else {
			m.Ack() // Acknowledge non-test messages
		}
	})

	// Either we get context canceled (because we found a message or timeout)
	// or we get a context deadline exceeded (expected when no messages)
	if err != nil && err != context.Canceled && !isContextDeadlineExceeded(err) {
		t.Fatalf("Unexpected error checking source subscription: %v", err)
	}

	if foundMessage {
		t.Fatalf("Expected no messages in source subscription, but found one")
	}

	t.Logf("Successfully verified DLR invalid input handling: 1 message moved after invalid input, 1 message discarded after multiple invalid inputs")
}

// Helper function to check if error is due to context deadline exceeded
func isContextDeadlineExceeded(err error) bool {
	return err.Error() == "context deadline exceeded"
}
