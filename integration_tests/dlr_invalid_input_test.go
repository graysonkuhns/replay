package cmd_test

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"testing"
	"time"

	"replay/integration_tests/testhelpers"

	"cloud.google.com/go/pubsub"
)

func TestDLRInvalidInputHandling(t *testing.T) {
	// Test to verify that the DLR operation correctly handles invalid input by asking again
	ctx := context.Background()
	projectID := os.Getenv("GCP_PROJECT")
	if projectID == "" {
		t.Fatal("GCP_PROJECT environment variable must be set")
	}

	// Define resource names.
	sourceTopicName := "default-events-dead-letter"
	sourceSubName := "default-events-dead-letter-subscription"
	destTopicName := "default-events"
	destSubName := "default-events-subscription"
	testRunValue := "dlr_invalid_input_test"

	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		t.Fatalf("Failed to create PubSub client: %v", err)
	}
	defer client.Close()

	// Purge subscriptions.
	sourceSub := client.Subscription(sourceSubName)
	log.Printf("Purging source subscription: %s", sourceSubName)
	if err := testhelpers.PurgeSubscription(ctx, sourceSub); err != nil {
		t.Fatalf("Failed to purge source subscription: %v", err)
	}
	destSub := client.Subscription(destSubName)
	destCtx, destCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer destCancel()
	if err := testhelpers.PurgeSubscription(destCtx, destSub); err != nil {
		t.Fatalf("Failed to purge destination subscription: %v", err)
	}

	// Prepare 2 messages with unique body content.
	numMessages := 2
	sourceTopic := client.Topic(sourceTopicName)
	var messages []pubsub.Message
	orderingKey := "test-ordering-key"
	
	for i := 1; i <= numMessages; i++ {
		body := fmt.Sprintf("DLR Invalid Input Test message %d", i)
		messages = append(messages, pubsub.Message{
			Data: []byte(body),
			Attributes: map[string]string{
				"testRun": testRunValue,
			},
		})
	}

	_, err = testhelpers.PublishTestMessages(ctx, sourceTopic, messages, orderingKey)
	if err != nil {
		t.Fatalf("Failed to publish test messages with ordering key: %v", err)
	}
	
	log.Printf("Published %d messages with ordering key: %s", numMessages, orderingKey)
	time.Sleep(15 * time.Second)  // Wait for messages to arrive in the subscription

	// Prepare CLI arguments for the dlr command.
	dlrArgs := []string{
		"dlr",
		"--source-type", "GCP_PUBSUB_SUBSCRIPTION",
		"--destination-type", "GCP_PUBSUB_TOPIC",
		"--source", fmt.Sprintf("projects/%s/subscriptions/%s", projectID, sourceSubName),
		"--destination", fmt.Sprintf("projects/%s/topics/%s", projectID, destTopicName),
	}

	// Simulate user inputs:
	// For message 1: invalid input "x", then "m" (move)
	// For message 2: invalid inputs "invalid", "123", then "d" (discard)
	origStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe for stdin: %v", err)
	}
	
	// Write inputs with intentional invalid entries
	inputs := "x\nm\ninvalid\n123\nd\n"
	
	_, err = io.WriteString(w, inputs)
	if err != nil {
		t.Fatalf("Failed to write simulated input: %v", err)
	}
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()

	// Run the dlr command.
	actual, err := testhelpers.RunCLICommand(dlrArgs)
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}
	
	// Define expected output substrings.
	expectedLines := []string{
		fmt.Sprintf("Starting DLR review from projects/%s/subscriptions/%s", projectID, sourceSubName),
		"",
		"Message 1:",
		"Data:",
		"DLR Invalid Input Test message 1",
		"Attributes: map[testRun:dlr_invalid_input_test]",
		"Choose action ([m]ove / [d]iscard / [q]uit): Invalid input. Please enter 'm', 'd', or 'q'.",
		"Choose action ([m]ove / [d]iscard / [q]uit): Message 1 moved successfully",
		"",
		"Message 2:",
		"Data:",
		"DLR Invalid Input Test message 2",
		"Attributes: map[testRun:dlr_invalid_input_test]",
		"Choose action ([m]ove / [d]iscard / [q]uit): Invalid input. Please enter 'm', 'd', or 'q'.",
		"Choose action ([m]ove / [d]iscard / [q]uit): Invalid input. Please enter 'm', 'd', or 'q'.",
		"Choose action ([m]ove / [d]iscard / [q]uit): Message 2 discarded (acked)",
		"",
		"Dead-lettered messages review completed. Total messages processed: 2",
	}
	
	testhelpers.AssertCLIOutput(t, actual, expectedLines)
	t.Logf("DLR command executed for invalid input handling test")

	// Allow time for moved messages to propagate.
	time.Sleep(5 * time.Second)

	// Poll the destination subscription for moved messages.
	// We expect exactly 1 message to be moved (message 1).
	received, err := testhelpers.PollMessages(ctx, destSub, testRunValue, 1)
	if err != nil {
		t.Fatalf("Error receiving messages from destination: %v", err)
	}
	if len(received) != 1 {
		t.Fatalf("Expected 1 message in destination, got %d", len(received))
	}
	
	// Verify the correct body of the moved message
	expectedMovedMessage := "DLR Invalid Input Test message 1"
	if string(received[0].Data) != expectedMovedMessage {
		t.Fatalf("Expected moved message body '%s', but got '%s'", 
			expectedMovedMessage, string(received[0].Data))
	}
	
	// Verify that no messages remain in the source subscription by using a custom checking approach
	// instead of using PollMessages which expects a specific number of messages
	time.Sleep(5 * time.Second)
	
	// Create a custom receiver function to check for any messages
	var foundMessage bool
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	// Use Receive directly instead of PollMessages
	err = sourceSub.Receive(cctx, func(ctx context.Context, m *pubsub.Message) {
		if m.Attributes["testRun"] == testRunValue {
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
