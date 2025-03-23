package cmd_test

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"testing"
	"time"

	"cloud.google.com/go/pubsub"

	"replay/cmd"
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
	// Here the move command will pull messages from the dead letter subscription and
	// publish them to the normal events topic.
	moveArgs := []string{
		"move",
		"--source-type", "GCP_PUBSUB_SUBSCRIPTION",
		"--destination-type", "GCP_PUBSUB_TOPIC",
		"--source", fmt.Sprintf("projects/%s/subscriptions/%s", projectID, sourceSubName),
		"--destination", fmt.Sprintf("projects/%s/topics/%s", projectID, destTopicName),
		"--count", fmt.Sprintf("%d", numMessages),
	}

	// Log before executing move command
	log.Printf("Executing move command with args: %v", moveArgs)
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = append([]string{"replay"}, moveArgs...)

	// Capture CLI output using os.Pipe.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	oldOut := os.Stdout
	os.Stdout = w
	// Run the move command.
	cmd.Execute()
	log.Printf("Move command executed")
	// Restore os.Stdout.
	w.Close()
	os.Stdout = oldOut

	// Read captured output.
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("Failed to read captured output: %v", err)
	}
	r.Close()

	// Replace timestamp parts with a token.
	actual := buf.String()
	tsRe := regexp.MustCompile(`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}`)
	actual = tsRe.ReplaceAllString(actual, "[TIMESTAMP]")

	// Define expected output with log lines included.
	expectedOutput := fmt.Sprintf(
		"[TIMESTAMP] Moving messages from projects/%s/subscriptions/%s to projects/%s/topics/%s\n"+
			"[TIMESTAMP] Pulled message 1\n[TIMESTAMP] Publishing message 1\n[TIMESTAMP] Published message 1 successfully\n[TIMESTAMP] Acked message 1\n[TIMESTAMP] Processed message 1\n"+
			"[TIMESTAMP] Pulled message 2\n[TIMESTAMP] Publishing message 2\n[TIMESTAMP] Published message 2 successfully\n[TIMESTAMP] Acked message 2\n[TIMESTAMP] Processed message 2\n"+
			"[TIMESTAMP] Pulled message 3\n[TIMESTAMP] Publishing message 3\n[TIMESTAMP] Published message 3 successfully\n[TIMESTAMP] Acked message 3\n[TIMESTAMP] Processed message 3\n"+
			"[TIMESTAMP] Move operation completed. Total messages moved: %d\n"+
			"[TIMESTAMP] Move command executed\n",
		projectID, sourceSubName, projectID, destTopicName, numMessages)

	if actual != expectedOutput {
		t.Fatalf("CLI output mismatch.\nExpected (with timestamps replaced):\n%q\nGot:\n%q", expectedOutput, actual)
	}

	// Allow time for messages to propagate to the destination.
	time.Sleep(5 * time.Second)
	log.Printf("Waiting for messages to propagate to destination subscription")

	// Pull moved messages from the destination subscription provided by Terraform.
	// For verification we assume the destination subscription does exist.
	// (If necessary, a separate subscription may be created in Terraform and referenced here.)
	// In this case we'll use the destination subscription "default-events-subscription".
	log.Printf("Starting to receive messages from destination subscription: default-events-subscription")
	destSub := client.Subscription("default-events-subscription")
	received := make([]*pubsub.Message, 0)
	cctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	err = destSub.Receive(cctx, func(ctx context.Context, m *pubsub.Message) {
		// Only count messages from our test run
		if m.Attributes["testRun"] == testRunValue {
			log.Printf("Received test message: %s", string(m.Data))
			received = append(received, m)
		} else {
			log.Printf("Ignoring non-test message: %s", string(m.Data))
		}
		m.Ack()
		if len(received) >= numMessages {
			cancel()
		}
	})
	if err != nil && err != context.Canceled {
		t.Fatalf("Error receiving messages from destination subscription: %v", err)
	}

	if len(received) != numMessages {
		t.Fatalf("Expected %d moved messages, got %d", numMessages, len(received))
	}

	log.Printf("Successfully received %d messages", len(received))
	t.Logf("Successfully moved %d messages", numMessages)
}
