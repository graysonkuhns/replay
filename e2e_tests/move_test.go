package cmd_test

import (
	"fmt"
	"testing"

	"replay/e2e_tests/testhelpers"
)

func init() {
	// Suppress logs to avoid interfering with parallel test output
	// log.Printf("Test suite initialization: logs are enabled")
}

func TestMoveStopsWhenSourceExhausted(t *testing.T) {
	t.Parallel()
	// Set up e2e test environment
	baseTest := testhelpers.NewBaseE2ETest(t, "move_test")
	// For this test, we move messages from the dead letter infrastructure to the normal events infrastructure.

	numMessages := 3
	messages := baseTest.CreateTestMessages(numMessages, "Test message")

	if err := baseTest.PublishAndWait(messages); err != nil {
		t.Fatalf("Failed to publish test messages: %v", err)
	}

	actual, err := baseTest.RunMoveCommand(0) // 0 means no count limit
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}
	// log.Printf("Move command executed")

	// Define expected output lines.
	expectedLines := []string{
		fmt.Sprintf("[TIMESTAMP] Moving messages from %s to %s", baseTest.Setup.GetSourceSubscriptionName(), baseTest.Setup.GetDestTopicName()),
		"[TIMESTAMP] Pulled message 1",
		"[TIMESTAMP] Publishing message 1",
		"[TIMESTAMP] Published message 1 successfully",
		"[TIMESTAMP] Acked message 1",
		"[TIMESTAMP] Processed message 1",
		"[TIMESTAMP] Pulled message 2",
		"[TIMESTAMP] Publishing message 2",
		"[TIMESTAMP] Published message 2 successfully",
		"[TIMESTAMP] Acked message 2",
		"[TIMESTAMP] Processed message 2",
		"[TIMESTAMP] Pulled message 3",
		"[TIMESTAMP] Publishing message 3",
		"[TIMESTAMP] Published message 3 successfully",
		"[TIMESTAMP] Acked message 3",
		"[TIMESTAMP] Processed message 3",
		fmt.Sprintf("[TIMESTAMP] Move operation completed. Total messages moved: %d", numMessages),
	}

	testhelpers.AssertCLIOutput(t, actual, expectedLines)

	baseTest.WaitForMessagePropagation()

	// Verify messages were moved to destination
	if err := baseTest.VerifyMessagesInDestination(numMessages); err != nil {
		t.Fatalf("%v", err)
	}

	t.Logf("Successfully moved %d messages", numMessages)
}

func TestMoveOperationWithCount(t *testing.T) {
	t.Parallel()
	// Set up e2e test environment
	baseTest := testhelpers.NewBaseE2ETest(t, "move_test_count")

	numMessages := 5
	moveCount := 3

	messages := baseTest.CreateTestMessages(numMessages, "Count Test message")

	if err := baseTest.PublishAndWait(messages); err != nil {
		t.Fatalf("Failed to publish test messages: %v", err)
	}

	actual, err := baseTest.RunMoveCommand(moveCount)
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}
	// log.Printf("Move command executed with count %d", moveCount)

	expectedLines := []string{
		fmt.Sprintf("[TIMESTAMP] Moving messages from %s to %s", baseTest.Setup.GetSourceSubscriptionName(), baseTest.Setup.GetDestTopicName()),
		"[TIMESTAMP] Pulled message 1",
		"[TIMESTAMP] Publishing message 1",
		"[TIMESTAMP] Published message 1 successfully",
		"[TIMESTAMP] Acked message 1",
		"[TIMESTAMP] Processed message 1",
		"[TIMESTAMP] Pulled message 2",
		"[TIMESTAMP] Publishing message 2",
		"[TIMESTAMP] Published message 2 successfully",
		"[TIMESTAMP] Acked message 2",
		"[TIMESTAMP] Processed message 2",
		"[TIMESTAMP] Pulled message 3",
		"[TIMESTAMP] Publishing message 3",
		"[TIMESTAMP] Published message 3 successfully",
		"[TIMESTAMP] Acked message 3",
		"[TIMESTAMP] Processed message 3",
		fmt.Sprintf("[TIMESTAMP] Move operation completed. Total messages moved: %d", moveCount),
	}

	testhelpers.AssertCLIOutput(t, actual, expectedLines)

	baseTest.WaitForMessagePropagation()

	// Verify the correct number of messages were moved to destination
	if err := baseTest.VerifyMessagesInDestination(moveCount); err != nil {
		t.Fatalf("%v", err)
	}

	// Verify the remaining messages are still in source
	if err := baseTest.VerifyMessagesInSource(numMessages - moveCount); err != nil {
		t.Fatalf("%v", err)
	}

	t.Logf("Successfully moved %d messages and found %d remaining in source", moveCount, numMessages-moveCount)
}

func TestMoveMessageBodyIntegrity(t *testing.T) {
	t.Parallel()
	// New test to verify that the body content of moved messages remains unchanged.
	baseTest := testhelpers.NewBaseE2ETest(t, "move_test_body_integrity")

	// Prepare messages with unique body content.
	numMessages := 3
	messages := baseTest.CreateTestMessages(numMessages, "Integrity Test message")
	var expectedBodies []string
	for _, msg := range messages {
		expectedBodies = append(expectedBodies, string(msg.Data))
	}

	if err := baseTest.PublishAndWait(messages); err != nil {
		t.Fatalf("Failed to publish test messages: %v", err)
	}

	// Run the move command.
	_, err := baseTest.RunMoveCommand(numMessages)
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}
	t.Logf("Move command executed for body integrity test")

	// Poll the destination subscription.
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
				// Use AssertMessageContent for consistency
				testhelpers.AssertMessageContent(t, string(msg.Data), expected)
				break
			}
		}
		if !found {
			t.Fatalf("Expected message body '%s' not found in received messages", expected)
		}
	}
	t.Logf("Message body integrity verified for all %d messages", numMessages)
}
