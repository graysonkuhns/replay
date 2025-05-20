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

func TestDLRMessageBodyIntegrity(t *testing.T) {
	// Test to verify that the body content of moved messages remains unchanged when using the DLR operation.
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
	testRunValue := "dlr_integrity_test"

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

	// Prepare messages with unique body content.
	numMessages := 3
	sourceTopic := client.Topic(sourceTopicName)
	var messages []pubsub.Message
	var expectedBodies []string

	for i := 1; i <= numMessages; i++ {
		body := fmt.Sprintf("DLR Integrity Test message %d", i)
		expectedBodies = append(expectedBodies, body)
		messages = append(messages, pubsub.Message{
			Data: []byte(body),
			Attributes: map[string]string{
				"testRun": testRunValue,
			},
		})
	}

	_, err = testhelpers.PublishTestMessages(ctx, sourceTopic, messages, "test-ordering-key")
	if err != nil {
		t.Fatalf("Failed to publish test messages: %v", err)
	}
	time.Sleep(15 * time.Second) // Wait for messages to arrive in the subscription

	// Prepare CLI arguments for the dlr command.
	dlrArgs := []string{
		"dlr",
		"--source-type", "GCP_PUBSUB_SUBSCRIPTION",
		"--destination-type", "GCP_PUBSUB_TOPIC",
		"--source", fmt.Sprintf("projects/%s/subscriptions/%s", projectID, sourceSubName),
		"--destination", fmt.Sprintf("projects/%s/topics/%s", projectID, destTopicName),
	}

	// Simulate user inputs: "m" (move) for all messages
	origStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe for stdin: %v", err)
	}

	// Write "m" for each message to move them all
	var inputs string
	for i := 0; i < numMessages; i++ {
		inputs += "m\n"
	}

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
		"DLR Integrity Test message 1",
		"Attributes: map[testRun:dlr_integrity_test]",
		"Choose action ([m]ove / [d]iscard / [q]uit): Message 1 moved successfully",
		"",
		"Message 2:",
		"Data:",
		"DLR Integrity Test message 2",
		"Attributes: map[testRun:dlr_integrity_test]",
		"Choose action ([m]ove / [d]iscard / [q]uit): Message 2 moved successfully",
		"",
		"Message 3:",
		"Data:",
		"DLR Integrity Test message 3",
		"Attributes: map[testRun:dlr_integrity_test]",
		"Choose action ([m]ove / [d]iscard / [q]uit): Message 3 moved successfully",
		"",
		"Dead-lettered messages review completed. Total messages processed: 3",
	}

	testhelpers.AssertCLIOutput(t, actual, expectedLines)
	t.Logf("DLR command executed for body integrity test")

	// Allow time for moved messages to propagate.
	time.Sleep(5 * time.Second)

	// Poll the destination subscription for moved messages.
	received, err := testhelpers.PollMessages(ctx, destSub, testRunValue, numMessages)
	if err != nil {
		t.Fatalf("Error receiving messages from destination: %v", err)
	}
	if len(received) != numMessages {
		t.Fatalf("Expected %d messages in destination, got %d", numMessages, len(received))
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
	sourceReceived, err := testhelpers.PollMessages(ctx, sourceSub, testRunValue, 0)
	if err != nil {
		t.Fatalf("Error polling source subscription: %v", err)
	}
	if len(sourceReceived) != 0 {
		t.Fatalf("Expected 0 messages in source subscription, got %d", len(sourceReceived))
	}

	t.Logf("Message body integrity verified for all %d messages moved using DLR operation", numMessages)
}
