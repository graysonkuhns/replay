package cmd_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/pubsub"

	"replay/integration_tests/testhelpers"
)

func init() {
	log.Printf("Test suite initialization: logs are enabled")
}

func TestMoveStopsWhenSourceExhausted(t *testing.T) {
	log.Printf("Starting TestMoveStopsWhenSourceExhausted: verifying stop when source runs out of messages")
	ctx := context.Background()
	projectID := os.Getenv("GCP_PROJECT")
	if projectID == "" {
		t.Fatal("GCP_PROJECT environment variable must be set")
	}
	// For this test, we move messages from the dead letter infrastructure to the normal events infrastructure.
	sourceTopicName := "default-events-dead-letter"
	sourceSubName := "default-events-dead-letter-subscription"
	destTopicName := "default-events"

	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		t.Fatalf("Failed to create PubSub client: %v", err)
	}
	defer client.Close()

	sourceSub := client.Subscription(sourceSubName)
	log.Printf("Purging source subscription: %s", sourceSubName)
	if err := testhelpers.PurgeSubscription(ctx, sourceSub); err != nil {
		t.Fatalf("Failed to purge source subscription: %v", err)
	}
	log.Printf("Completed purge of source subscription: %s", sourceSubName)

	log.Printf("Purging destination subscription: default-events-subscription")
	destSubPurge := client.Subscription("default-events-subscription")
	destCtx, destCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer destCancel()
	if err := testhelpers.PurgeSubscription(destCtx, destSubPurge); err != nil {
		t.Fatalf("Failed to purge destination subscription: %v", err)
	}
	log.Printf("Completed purge of destination subscription: default-events-subscription")

	sourceTopic := client.Topic(sourceTopicName)
	numMessages := 3
	testRunValue := "move_test"

	var messages []pubsub.Message
	for i := 1; i <= numMessages; i++ {
		messages = append(messages, pubsub.Message{
			Data: []byte(fmt.Sprintf("Test message %d", i)),
			Attributes: map[string]string{
				"testRun": testRunValue,
			},
		})
	}

	_, err = testhelpers.PublishTestMessages(ctx, sourceTopic, messages)
	if err != nil {
		t.Fatalf("Failed to publish test messages: %v", err)
	}

	time.Sleep(10 * time.Second)
	log.Printf("Completed waiting for dead letter subscription to receive messages")

	moveArgs := []string{
		"move",
		"--source-type", "GCP_PUBSUB_SUBSCRIPTION",
		"--destination-type", "GCP_PUBSUB_TOPIC",
		"--source", fmt.Sprintf("projects/%s/subscriptions/%s", projectID, sourceSubName),
		"--destination", fmt.Sprintf("projects/%s/topics/%s", projectID, destTopicName),
	}

	actual, err := testhelpers.RunCLICommand(moveArgs)
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}
	log.Printf("Move command executed")

	// Define expected output lines.
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
		"[TIMESTAMP] No messages received within timeout",
		fmt.Sprintf("[TIMESTAMP] Move operation completed. Total messages moved: %d", numMessages),
	}

	testhelpers.AssertCLIOutput(t, actual, expectedLines)

	time.Sleep(5 * time.Second)
	log.Printf("Waiting for messages to propagate to destination subscription")

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

func TestMoveOperationWithCount(t *testing.T) {
	log.Printf("Starting TestMoveOperationWithCount")
	ctx := context.Background()
	projectID := os.Getenv("GCP_PROJECT")
	if projectID == "" {
		t.Fatal("GCP_PROJECT environment variable must be set")
	}

	// Define test parameters.
	sourceTopicName := "default-events-dead-letter"
	sourceSubName := "default-events-dead-letter-subscription"
	destTopicName := "default-events"
	destSubName := "default-events-subscription"

	// Create client and purge subscriptions.
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		t.Fatalf("Failed to create PubSub client: %v", err)
	}
	defer client.Close()

	sourceSub := client.Subscription(sourceSubName)
	if err := testhelpers.PurgeSubscription(ctx, sourceSub); err != nil {
		t.Fatalf("Failed to purge source subscription: %v", err)
	}
	destSub := client.Subscription(destSubName)
	destCtx, destCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer destCancel()
	if err := testhelpers.PurgeSubscription(destCtx, destSub); err != nil {
		t.Fatalf("Failed to purge destination subscription: %v", err)
	}

	numMessages := 5
	moveCount := 3
	testRunValue := "move_test_count"
	sourceTopic := client.Topic(sourceTopicName)

	var messages []pubsub.Message
	for i := 1; i <= numMessages; i++ {
		messages = append(messages, pubsub.Message{
			Data: []byte(fmt.Sprintf("Count Test message %d", i)),
			Attributes: map[string]string{
				"testRun": testRunValue,
			},
		})
	}

	_, err = testhelpers.PublishTestMessages(ctx, sourceTopic, messages)
	if err != nil {
		t.Fatalf("Failed to publish test messages: %v", err)
	}

	time.Sleep(10 * time.Second)
	log.Printf("Completed waiting for messages to be available in the source subscription")

	moveArgs := []string{
		"move",
		"--source-type", "GCP_PUBSUB_SUBSCRIPTION",
		"--destination-type", "GCP_PUBSUB_TOPIC",
		"--source", fmt.Sprintf("projects/%s/subscriptions/%s", projectID, sourceSubName),
		"--destination", fmt.Sprintf("projects/%s/topics/%s", projectID, destTopicName),
		"--count", fmt.Sprintf("%d", moveCount),
	}

	actual, err := testhelpers.RunCLICommand(moveArgs)
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}
	log.Printf("Move command executed with count %d", moveCount)

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
		fmt.Sprintf("[TIMESTAMP] Move operation completed. Total messages moved: %d", moveCount),
	}

	testhelpers.AssertCLIOutput(t, actual, expectedLines)

	time.Sleep(5 * time.Second)
	log.Printf("Polling destination subscription for moved messages")
	movedMessages, err := testhelpers.PollMessages(ctx, destSub, testRunValue, moveCount)
	if err != nil {
		t.Fatalf("Error receiving moved messages: %v", err)
	}
	if len(movedMessages) != moveCount {
		t.Fatalf("Expected %d moved messages in destination, got %d", moveCount, len(movedMessages))
	}

	log.Printf("Polling source subscription for remaining messages")
	remainingMessages, err := testhelpers.PollMessages(ctx, sourceSub, testRunValue, numMessages-moveCount)
	if err != nil {
		t.Fatalf("Error receiving remaining messages: %v", err)
	}
	if len(remainingMessages) != numMessages-moveCount {
		t.Fatalf("Expected %d remaining messages in source, got %d", numMessages-moveCount, len(remainingMessages))
	}

	t.Logf("Successfully moved %d messages and found %d remaining in source", moveCount, numMessages-moveCount)
}
