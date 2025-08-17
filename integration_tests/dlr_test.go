package cmd_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"replay/integration_tests/testhelpers"

	"cloud.google.com/go/pubsub/v2"
)

func TestDLROperation(t *testing.T) {
	t.Parallel()
	// Set up integration test environment
	setup := testhelpers.SetupIntegrationTest(t)
	testRunValue := "dlr_test"

	// Publish two test messages to the dead-letter topic: one to move and one to discard.
	sourceTopicName := setup.GetSourceTopicName()
	messages := []pubsub.Message{
		{
			Data: []byte("Test message move"),
			Attributes: map[string]string{
				"testRun": testRunValue,
			},
		},
		{
			Data: []byte("Test message discard"),
			Attributes: map[string]string{
				"testRun": testRunValue,
			},
		},
	}
	_, err := testhelpers.PublishTestMessages(setup.Context, setup.Client, sourceTopicName, messages, "test-ordering-key")
	if err != nil {
		t.Fatalf("Failed to publish test messages: %v", err)
	}

	// Wait for the messages to propagate to the dead-letter subscription.
	time.Sleep(20 * time.Second)

	// Prepare CLI arguments for the dlr command.
	dlrArgs := []string{
		"dlr",
		"--source-type", "GCP_PUBSUB_SUBSCRIPTION",
		"--destination-type", "GCP_PUBSUB_TOPIC",
		"--source", setup.GetSourceSubscriptionName(),
		"--destination", setup.GetDestTopicName(),
	}

	// Simulate user inputs: "m" for moving message 1 and "d" for discarding message 2.
	simulator, err := testhelpers.NewStdinSimulator("m\nd\n")
	if err != nil {
		t.Fatalf("Failed to create stdin simulator: %v", err)
	}
	defer simulator.Cleanup()

	// Run the dlr command.
	actual, err := testhelpers.RunCLICommand(dlrArgs)
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}

	// Verify the output structure without assuming message order
	if !strings.Contains(actual, fmt.Sprintf("Starting DLR review from %s", setup.GetSourceSubscriptionName())) {
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
	time.Sleep(20 * time.Second)

	// Verify the message was published to the destination topic via its subscription.
	received, err := testhelpers.PollMessages(setup.Context, setup.Client, setup.GetDestSubscriptionName(), testRunValue, 1)
	if err != nil {
		t.Fatalf("Error receiving messages: %v", err)
	}
	if len(received) != 1 {
		t.Fatalf("Expected 1 moved message, got %d", len(received))
	}
	// Suppress logs to avoid interfering with parallel test output
	// log.Printf("Successfully moved and received %d message", len(received))

	// Verify that the discarded message is no longer in the source subscription.
	sourceReceived, err := testhelpers.PollMessages(setup.Context, setup.Client, setup.GetSourceSubscriptionName(), testRunValue, 0)
	if err != nil {
		t.Fatalf("Error polling source subscription: %v", err)
	}
	if len(sourceReceived) != 0 {
		t.Fatalf("Expected 0 messages in source subscription, got %d", len(sourceReceived))
	}
}
