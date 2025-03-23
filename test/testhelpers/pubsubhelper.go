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

// PublishTestMessages publishes a slice of test messages to the given topic.
func PublishTestMessages(ctx context.Context, topic *pubsub.Topic, messages []pubsub.Message) ([]string, error) {
	var publishIDs []string
	for i, msg := range messages {
		log.Printf("Publishing message %d", i+1)
		result := topic.Publish(ctx, &msg)
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
