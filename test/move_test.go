package cmd_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings" // added import for joining expected output lines
	"testing"
	"time"

	"cloud.google.com/go/pubsub"

	"replay/test/testhelpers" // added helper import
	// updated import: using new package
)

// Added init function to log at startup
func init() {
	log.Printf("Test suite initialization: logs are enabled")
}

func TestMoveOperation(t *testing.T) {
	log.Printf("Starting TestMoveOperation")
	ctx := context.Background()
	projectID := os.Getenv("GCP_PROJECT")
	if projectID == "" {
		t.Fatal("GCP_PROJECT environment variable must be set")
	}
	// For this test, we move messages from the dead letter infra to the normal events infra.
	// Use dead letter topic/subscription as source...
	sourceTopicName := "default-events-dead-letter"
	sourceSubName := "default-events-dead-letter-subscription"
	// ...and use normal events topic as destination.
	destTopicName := "default-events"
	// Reference the already created destination subscription if needed for validation,
	// here we use the destination topic and the dead letter subscription is now our source.

	// create a Pub/Sub client
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		t.Fatalf("Failed to create PubSub client: %v", err)
	}
	defer client.Close()

	// reference source subscription
	sourceSub := client.Subscription(sourceSubName)

	// Log before purging the source subscription
	log.Printf("Purging source subscription: %s", sourceSubName)
	if err := testhelpers.PurgeSubscription(ctx, sourceSub); err != nil {
		t.Fatalf("Failed to purge source subscription: %v", err)
	}
	log.Printf("Completed purge of source subscription: %s", sourceSubName)

	// Purge destination subscription too.
	log.Printf("Purging destination subscription: default-events-subscription")
	destSubPurge := client.Subscription("default-events-subscription")
	destCtx, destCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer destCancel()
	if err := testhelpers.PurgeSubscription(destCtx, destSubPurge); err != nil {
		t.Fatalf("Failed to purge destination subscription: %v", err)
	}
	log.Printf("Completed purge of destination subscription: default-events-subscription")

	// publish some test messages to the dead letter topic (source topic).
	sourceTopic := client.Topic(sourceTopicName)
	numMessages := 3
	testRunValue := "move_test" // marker for messages

	// Define test messages.
	var messages []pubsub.Message
	for i := 1; i <= numMessages; i++ {
		messages = append(messages, pubsub.Message{
			Data: []byte(fmt.Sprintf("Test message %d", i)),
			Attributes: map[string]string{
				"testRun": testRunValue,
			},
		})
	}

	// Call helper to log and publish test messages.
	_, err = testhelpers.PublishTestMessages(ctx, sourceTopic, messages)
	if err != nil {
		t.Fatalf("Failed to publish test messages: %v", err)
	}

	// Allow time for the dead letter subscription to receive the published messages.
	time.Sleep(10 * time.Second)
	log.Printf("Completed waiting for dead letter subscription to receive messages")

	// Set up the CLI command arguments for the move operation.
	moveArgs := []string{
		"move",
		"--source-type", "GCP_PUBSUB_SUBSCRIPTION",
		"--destination-type", "GCP_PUBSUB_TOPIC",
		"--source", fmt.Sprintf("projects/%s/subscriptions/%s", projectID, sourceSubName),
		"--destination", fmt.Sprintf("projects/%s/topics/%s", projectID, destTopicName),
		"--count", fmt.Sprintf("%d", numMessages),
	}

	// Run CLI command using test helper.
	actual, err := testhelpers.RunCLICommand(moveArgs)
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}
	log.Printf("Move command executed")

	// Define expected output with each line on its own code line.
	expectedLines := []string{
		fmt.Sprintf("[TIMESTAMP] Moving messages from projects/%s/subscriptions/%s to projects/%s/topics/%s", projectID, sourceSubName, projectID, destTopicName),
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
		"[TIMESTAMP] Move command executed",
	}
	expectedOutput := strings.Join(expectedLines, "\n") + "\n"

	if actual != expectedOutput {
		// Split actual output into individual lines.
		actualLines := strings.Split(strings.TrimSpace(actual), "\n")
		// Format expected and actual outputs to show one line per code line.
		expectedStr := strings.Join(expectedLines, "\n")
		actualStr := strings.Join(actualLines, "\n")
		t.Fatalf("CLI output mismatch.\nExpected output:\n%s\nActual output:\n%s", expectedStr, actualStr)
	}

	// Allow time for messages to propagate to the destination.
	time.Sleep(5 * time.Second)
	log.Printf("Waiting for messages to propagate to destination subscription")

	// Pull moved messages from the destination subscription using the helper.
	log.Printf("Starting to receive messages from destination subscription: default-events-subscription")
	destSub := client.Subscription("default-events-subscription")
	received, err := testhelpers.PollMessages(ctx, destSub, testRunValue, numMessages)
	if err != nil {
		t.Fatalf("Error receiving messages: %v", err)
	}
	log.Printf("Successfully received %d messages", len(received))

	if len(received) != numMessages {
		t.Fatalf("Expected %d moved messages, got %d", numMessages, len(received))
	}

	log.Printf("Successfully received %d messages", len(received))
	t.Logf("Successfully moved %d messages", numMessages)
}
