package testhelpers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"cloud.google.com/go/pubsub"
	pubsubapiv1 "cloud.google.com/go/pubsub/apiv1"
	pubsubpb "cloud.google.com/go/pubsub/apiv1/pubsubpb"
)

// PurgeSubscription purges messages from the given Pub/Sub subscription.
func PurgeSubscription(ctx context.Context, sub *pubsub.Subscription) error {
	subResource := sub.String() // assumes full resource name (e.g. projects/<proj>/subscriptions/<sub>)
	subscriberClient, err := pubsubapiv1.NewSubscriberClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create subscriber client: %w", err)
	}
	defer subscriberClient.Close()

	for {
		pollCtx, pollCancel := context.WithTimeout(ctx, 5*time.Second)
		req := &pubsubpb.PullRequest{
			Subscription: subResource,
			MaxMessages:  1,
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
		ackReq := &pubsubpb.AcknowledgeRequest{
			Subscription: subResource,
			AckIds:       []string{resp.ReceivedMessages[0].AckId},
		}
		if err := subscriberClient.Acknowledge(ctx, ackReq); err != nil {
			return fmt.Errorf("failed to acknowledge message: %w", err)
		}
	}
	return nil
}

func PublishTestMessages(ctx context.Context, topic *pubsub.Topic, messages []pubsub.Message, orderingKey string) ([]string, error) {
	var publishIDs []string
	
	// If ordering key is provided, enable message ordering on the topic
	if orderingKey != "" {
		topic.EnableMessageOrdering = true
	}
	
	for i, msg := range messages {
		msgToPublish := &msg // Use the original message by default
		
		// If ordering key is provided, create a copy with the ordering key
		if orderingKey != "" {
			log.Printf("Publishing message %d with ordering key: %s", i+1, orderingKey)
			
			// Create a new message with the ordering key
			msgToPublish = &pubsub.Message{
				Data:        msg.Data,
				Attributes:  msg.Attributes,
				OrderingKey: orderingKey,
			}
		} else {
			log.Printf("Publishing message %d", i+1)
		}
		
		result := topic.Publish(ctx, msgToPublish)
		id, err := result.Get(ctx)
		if err != nil {
			log.Printf("Failed to publish message %d: %v", i+1, err)
			return publishIDs, fmt.Errorf("failed to publish message %d: %w", i+1, err)
		}
		log.Printf("Published message %d with id: %s", i+1, id)
		publishIDs = append(publishIDs, id)
	}
	return publishIDs, nil
}

// PollMessages polls messages from a subscription and verifies the expected count.
func PollMessages(ctx context.Context, sub *pubsub.Subscription, testRunValue string, expectedCount int) ([]*pubsub.Message, error) {
	var received []*pubsub.Message
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	err := sub.Receive(cctx, func(ctx context.Context, m *pubsub.Message) {
		if m.Attributes["testRun"] == testRunValue {
			log.Printf("Received test message: %s", string(m.Data))
			received = append(received, m)
		} else {
			log.Printf("Ignoring non-test message: %s", string(m.Data))
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
