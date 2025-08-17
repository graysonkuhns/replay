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

func TestDLRQuitOperation(t *testing.T) {
	t.Parallel()
	// Test to verify that the DLR operation correctly handles the quit command
	baseTest := testhelpers.NewBaseIntegrationTest(t, "dlr_quit_test")

	// Purge any existing messages from the source subscription to ensure test isolation
	if err := testhelpers.PurgeSubscription(baseTest.Setup.Context, baseTest.Setup.Client, baseTest.Setup.GetSourceSubscriptionName()); err != nil {
		t.Fatalf("Failed to purge source subscription: %v", err)
	}

	// Prepare 4 messages with unique body content.
	numMessages := 4
	messages := baseTest.CreateTestMessages(numMessages, "DLR Quit Test message")

	if err := baseTest.PublishAndWait(messages); err != nil {
		t.Fatalf("Failed to publish test messages: %v", err)
	}

	// Add extra wait to ensure messages are properly available in subscription
	// This helps with test isolation when running in parallel
	time.Sleep(10 * time.Second)

	// Simulate user inputs: "m" (move) for 2 messages, "d" (discard) for 1 message, then "q" (quit)
	inputs := "m\nm\nd\nq\n"

	// Run the dlr command.
	actual, err := baseTest.RunDLRCommand(inputs)
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}

	// Instead of checking exact order, verify the structure and key operations
	// We expect 4 messages to be presented, with actions: move, move, discard, quit
	if !strings.Contains(actual, fmt.Sprintf("Starting DLR review from %s", baseTest.Setup.GetSourceSubscriptionName())) {
		t.Fatalf("Expected DLR start message not found in output")
	}

	// Count occurrences of each action result
	movedCount := strings.Count(actual, "moved successfully")
	discardedCount := strings.Count(actual, "discarded (acked)")
	quitCount := strings.Count(actual, "Quitting review...")

	// Log the actual output for debugging
	t.Logf("DLR output contained %d 'moved successfully', %d 'discarded (acked)', %d 'Quitting review...' messages",
		movedCount, discardedCount, quitCount)

	// Count how many of our test messages appear in the output
	testMessageCount := 0
	for i := 1; i <= 4; i++ {
		if strings.Contains(actual, fmt.Sprintf("DLR Quit Test message %d", i)) {
			testMessageCount++
		}
	}
	t.Logf("Found %d test messages in output", testMessageCount)

	if movedCount != 2 {
		t.Fatalf("Expected 2 'moved successfully' messages, got %d. Full output:\n%s", movedCount, actual)
	}
	if discardedCount != 1 {
		t.Fatalf("Expected 1 'discarded (acked)' message, got %d", discardedCount)
	}
	if quitCount != 1 {
		t.Fatalf("Expected 1 'Quitting review...' message, got %d", quitCount)
	}

	// Verify final summary
	if !strings.Contains(actual, "Dead-lettered messages review completed. Total messages processed: 3") {
		t.Fatalf("Expected summary with 3 processed messages not found")
	}

	// Verify all 4 test messages appear in the output (in any order)
	for i := 1; i <= 4; i++ {
		expectedMsg := fmt.Sprintf("DLR Quit Test message %d", i)
		if !strings.Contains(actual, expectedMsg) {
			t.Fatalf("Expected message '%s' not found in output", expectedMsg)
		}
	}
	t.Logf("DLR command executed for quit operation test")

	// Allow time for moved messages to propagate.
	baseTest.WaitForMessagePropagation()

	// Poll the destination subscription for moved messages.
	// We expect exactly 2 messages to be moved.
	received, err := baseTest.GetMessagesFromDestination(2)
	if err != nil {
		t.Fatalf("Error receiving messages from destination: %v", err)
	}

	// Verify that moved messages are from our test set
	for _, msg := range received {
		msgData := string(msg.Data)
		if !strings.HasPrefix(msgData, "DLR Quit Test message") {
			t.Fatalf("Unexpected message in destination: %s", msgData)
		}
	}

	// Wait for ack deadline to expire (60 seconds) before checking the source subscription
	time.Sleep(70 * time.Second)

	// Verify that one message remains in the source subscription
	// We expect exactly 1 message to remain in the source subscription after processing.
	// Try multiple times to account for timing variations when running with other tests
	var sourceReceived []*pubsub.Message
	maxAttempts := 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Use a longer timeout context for this specific polling operation
		pollCtx, pollCancel := context.WithTimeout(baseTest.Setup.Context, 60*time.Second)
		sourceReceived, err = testhelpers.PollMessages(pollCtx, baseTest.Setup.Client, baseTest.Setup.GetSourceSubscriptionName(), baseTest.TestRunID, 1)
		pollCancel()

		if err == nil && len(sourceReceived) == 1 {
			break // Success
		}

		if attempt < maxAttempts {
			t.Logf("Attempt %d: Expected 1 message in source, got %d. Retrying...", attempt, len(sourceReceived))
			time.Sleep(10 * time.Second)
		}
	}

	if err != nil {
		t.Fatalf("Error polling source subscription: %v", err)
	}
	if len(sourceReceived) != 1 {
		// Log more details to help debug the issue
		t.Logf("Test isolation issue: Expected 1 message in source subscription, got %d", len(sourceReceived))
		t.Logf("This test expects the 4th message to remain after quit, but it may have been processed")
		t.Fatalf("Expected 1 message in source subscription, got %d", len(sourceReceived))
	}

	// Verify the remaining message is from our test set
	remainingMsg := string(sourceReceived[0].Data)
	if !strings.HasPrefix(remainingMsg, "DLR Quit Test message") {
		t.Fatalf("Unexpected remaining message: %s", remainingMsg)
	}

	t.Logf("Successfully verified DLR quit operation: 2 messages moved, 1 discarded, 1 remaining after quit")
}
