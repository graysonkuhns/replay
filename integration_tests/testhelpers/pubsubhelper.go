package testhelpers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"cloud.google.com/go/pubsub/v2"
	pubsubpb "cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
)

// PurgeSubscription purges messages from the given Pub/Sub subscription.
func PurgeSubscription(ctx context.Context, client *pubsub.Client, subscriptionName string) error {
	subscriberClient := client.SubscriptionAdminClient
	// Don't close the client as it's shared across tests

	// Pull and ack messages in batches for better performance
	for {
		pollCtx, pollCancel := context.WithTimeout(ctx, 5*time.Second)
		req := &pubsubpb.PullRequest{
			Subscription: subscriptionName,
			MaxMessages:  100, // Pull up to 100 messages at a time
		}
		resp, err := subscriberClient.Pull(pollCtx, req)
		pollCancel()
		if err != nil {
			// assume timeout means no more messages available
			if strings.Contains(err.Error(), "DeadlineExceeded") {
				break
			}
			return fmt.Errorf("error during pull: %w", err)
		}
		if len(resp.ReceivedMessages) == 0 {
			break
		}

		// Collect all ack IDs
		var ackIds []string
		for _, msg := range resp.ReceivedMessages {
			ackIds = append(ackIds, msg.AckId)
		}

		// Ack all messages at once
		ackReq := &pubsubpb.AcknowledgeRequest{
			Subscription: subscriptionName,
			AckIds:       ackIds,
		}
		if err := subscriberClient.Acknowledge(ctx, ackReq); err != nil {
			return fmt.Errorf("failed to acknowledge messages: %w", err)
		}
	}
	return nil
}

func PublishTestMessages(ctx context.Context, client *pubsub.Client, topicName string, messages []pubsub.Message, orderingKey string) ([]string, error) {
	var publishIDs []string

	// Create publisher for the topic
	publisher := client.Publisher(topicName)
	// Only enable message ordering if an ordering key is provided
	if orderingKey != "" {
		publisher.EnableMessageOrdering = true
	}
	defer publisher.Stop()

	for i, msg := range messages {
		msgToPublish := &msg // Use the original message by default

		// If ordering key is provided, create a copy with the ordering key
		if orderingKey != "" {
			// Suppress logs to avoid interfering with parallel test output
			// log.Printf("Publishing message %d with ordering key: %s", i+1, orderingKey)

			// Create a new message with the ordering key
			msgToPublish = &pubsub.Message{
				Data:        msg.Data,
				Attributes:  msg.Attributes,
				OrderingKey: orderingKey,
			}
		}

		result := publisher.Publish(ctx, msgToPublish)
		id, err := result.Get(ctx)
		if err != nil {
			// Keep error logs as they are important for debugging
			log.Printf("Failed to publish message %d: %v", i+1, err)
			return publishIDs, fmt.Errorf("failed to publish message %d: %w", i+1, err)
		}
		// Suppress success logs to avoid interfering with parallel test output
		// log.Printf("Published message %d with id: %s", i+1, id)
		publishIDs = append(publishIDs, id)
	}
	return publishIDs, nil
}

// PollMessages polls messages from a subscription and verifies the expected count.
func PollMessages(ctx context.Context, client *pubsub.Client, subscriptionName string, testRunValue string, expectedCount int) ([]*pubsub.Message, error) {
	var received []*pubsub.Message
	cctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	subscriber := client.Subscriber(subscriptionName)
	err := subscriber.Receive(cctx, func(ctx context.Context, m *pubsub.Message) {
		if m.Attributes["testRun"] == testRunValue {
			// Suppress logs to avoid interfering with parallel test output
			// log.Printf("Received test message: %s", string(m.Data))
			received = append(received, m)
		}
		m.Ack()
		if len(received) >= expectedCount {
			cancel()
		}
	})
	if err != nil && err != context.Canceled {
		return nil, err
	}
	if len(received) != expectedCount {
		return received, fmt.Errorf("expected %d messages, got %d", expectedCount, len(received))
	}
	return received, nil
}
