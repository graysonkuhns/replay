package cmd_test

import (
	"cloud.google.com/go/pubsub"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"replay/cmd" // import the CLI package to get rootCmd
)

// Added init function to log at startup
func init() {
	log.Printf("Test suite initialization: logs are enabled")
}

// helper to purge a subscription by pulling (and acking) all available messages.
func purgeSubscription(ctx context.Context, sub *pubsub.Subscription) error {
	// pull with short timeout repeatedly until no message is received.
	for {
		cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		err := sub.Receive(cctx, func(ctx context.Context, m *pubsub.Message) {
			m.Ack()
		})
		// If error contains "context deadline exceeded" assume no more messages.
		if err != nil && strings.Contains(err.Error(), "context deadline exceeded") {
			break
		} else if err != nil {
			return fmt.Errorf("failed during purge: %w", err)
		}
	}
	return nil
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
	if err := purgeSubscription(ctx, sourceSub); err != nil {
		t.Fatalf("Failed to purge source subscription: %v", err)
	}
	log.Printf("Completed purge of source subscription: %s", sourceSubName)

	// publish some test messages to the dead letter topic (source topic).
	sourceTopic := client.Topic(sourceTopicName)
	var publishIDs []string
	numMessages := 3

	// Log before publishing test messages
	log.Printf("Publishing %d test messages to dead letter topic: %s", numMessages, sourceTopicName)
	for i := 1; i <= numMessages; i++ {
		result := sourceTopic.Publish(ctx, &pubsub.Message{
			Data: []byte(fmt.Sprintf("Test message %d", i)),
		})
		id, err := result.Get(ctx)
		if err != nil {
			t.Fatalf("Failed to publish message %d: %v", i, err)
		}
		publishIDs = append(publishIDs, id)
		log.Printf("Published message %d with id: %s", i, id)
	}
	log.Printf("Published test messages with ids: %v", publishIDs)

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

	// Run the move command.
	cmd.Execute()
	log.Printf("Move command executed")

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
		log.Printf("Received message: %s", string(m.Data))
		received = append(received, m)
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