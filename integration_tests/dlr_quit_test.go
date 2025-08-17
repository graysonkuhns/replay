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
	setup := testhelpers.SetupIntegrationTest(t)
	testRunValue := "dlr_quit_test"

	// Purge any existing messages from the source subscription to ensure test isolation
	if err := testhelpers.PurgeSubscription(setup.Context, setup.Client, setup.GetSourceSubscriptionName()); err != nil {
		t.Fatalf("Failed to purge source subscription: %v", err)
	}

	// Prepare 4 messages with unique body content.
	numMessages := 4
	sourceTopicName := setup.GetSourceTopicName()
	var messages []pubsub.Message
	orderingKey := "test-ordering-key"

	for i := 1; i <= numMessages; i++ {
		body := fmt.Sprintf("DLR Quit Test message %d", i)
		messages = append(messages, pubsub.Message{
			Data: []byte(body),
			Attributes: map[string]string{
				"testRun": testRunValue,
			},
		})
	}

	_, err := testhelpers.PublishTestMessages(setup.Context, setup.Client, sourceTopicName, messages, orderingKey)
	if err != nil {
		t.Fatalf("Failed to publish test messages with ordering key: %v", err)
	}

	// Suppress logs to avoid interfering with parallel test output
	// log.Printf("Published %d messages with ordering key: %s", numMessages, orderingKey)
	time.Sleep(30 * time.Second) // Wait for messages to arrive in the subscription

	// Prepare CLI arguments for the dlr command.
	dlrArgs := []string{
		"dlr",
		"--source-type", "GCP_PUBSUB_SUBSCRIPTION",
		"--destination-type", "GCP_PUBSUB_TOPIC",
		"--source", setup.GetSourceSubscriptionName(),
		"--destination", setup.GetDestTopicName(),
	}

	// Simulate user inputs: "m" (move) for 2 messages, "d" (discard) for 1 message, then "q" (quit)
	inputs := "m\nm\nd\nq\n"
	simulator, err := testhelpers.NewStdinSimulator(inputs)
	if err != nil {
		t.Fatalf("Failed to create stdin simulator: %v", err)
	}
	defer simulator.Cleanup()

	// Run the dlr command.
	actual, err := testhelpers.RunCLICommand(dlrArgs)
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}

	// Instead of checking exact order, verify the structure and key operations
	// We expect 4 messages to be presented, with actions: move, move, discard, quit
	if !strings.Contains(actual, fmt.Sprintf("Starting DLR review from %s", setup.GetSourceSubscriptionName())) {
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
	time.Sleep(20 * time.Second)

	// Poll the destination subscription for moved messages.
	// We expect exactly 2 messages to be moved.
	received, err := testhelpers.PollMessages(setup.Context, setup.Client, setup.GetDestSubscriptionName(), testRunValue, 2)
	if err != nil {
		t.Fatalf("Error receiving messages from destination: %v", err)
	}
	if len(received) != 2 {
		t.Fatalf("Expected 2 messages in destination, got %d", len(received))
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
	// Use a longer timeout context for this specific polling operation
	pollCtx, pollCancel := context.WithTimeout(setup.Context, 60*time.Second)
	defer pollCancel()
	sourceReceived, err := testhelpers.PollMessages(pollCtx, setup.Client, setup.GetSourceSubscriptionName(), testRunValue, 1)

	if err != nil {
		t.Fatalf("Error polling source subscription: %v", err)
	}
	if len(sourceReceived) != 1 {
		t.Fatalf("Expected 1 message in source subscription, got %d", len(sourceReceived))
	}

	// Verify the remaining message is from our test set
	remainingMsg := string(sourceReceived[0].Data)
	if !strings.HasPrefix(remainingMsg, "DLR Quit Test message") {
		t.Fatalf("Unexpected remaining message: %s", remainingMsg)
	}

	t.Logf("Successfully verified DLR quit operation: 2 messages moved, 1 discarded, 1 remaining after quit")
}
