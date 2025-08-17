package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/pubsub/v2"
	pubsubpb "cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
)

// Message represents a message with its data and metadata
type Message struct {
	Data       []byte
	Attributes map[string]string
	AckID      string
}

// PullConfig contains configuration for message pulling
type PullConfig struct {
	MaxMessages int
	Timeout     time.Duration
}

// MessageBroker defines the interface for message operations
type MessageBroker interface {
	Pull(ctx context.Context, config PullConfig) (*Message, error)
	Publish(ctx context.Context, message *Message) error
	Acknowledge(ctx context.Context, ackID string) error
	Close() error
}

// PubSubBroker implements MessageBroker for Google Cloud Pub/Sub
type PubSubBroker struct {
	subClient    *pubsub.Client
	topicClient  *pubsub.Client
	publisher    *pubsub.Publisher
	subscription string
	topic        string
}

// NewPubSubBroker creates a new PubSubBroker
func NewPubSubBroker(ctx context.Context, subscription, topic string) (*PubSubBroker, error) {
	// Parse subscription project
	subParts := strings.Split(subscription, "/")
	if len(subParts) < 4 {
		return nil, fmt.Errorf("invalid subscription resource format: %s", subscription)
	}
	subProj := subParts[1]

	// Parse topic project
	topicParts := strings.Split(topic, "/")
	if len(topicParts) < 4 {
		return nil, fmt.Errorf("invalid topic resource format: %s", topic)
	}
	topicProj := topicParts[1]

	// Create subscription client
	subClient, err := pubsub.NewClient(ctx, subProj)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription client: %w", err)
	}

	// Create topic client (reuse if same project)
	var topicClient *pubsub.Client
	if topicProj == subProj {
		topicClient = subClient
	} else {
		topicClient, err = pubsub.NewClient(ctx, topicProj)
		if err != nil {
			subClient.Close()
			return nil, fmt.Errorf("failed to create topic client: %w", err)
		}
	}

	publisher := topicClient.Publisher(topic)

	return &PubSubBroker{
		subClient:    subClient,
		topicClient:  topicClient,
		publisher:    publisher,
		subscription: subscription,
		topic:        topic,
	}, nil
}

// Pull retrieves a single message from the subscription
func (b *PubSubBroker) Pull(ctx context.Context, config PullConfig) (*Message, error) {
	pullCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	req := &pubsubpb.PullRequest{
		Subscription: b.subscription,
		MaxMessages:  int32(config.MaxMessages),
	}

	resp, err := b.subClient.SubscriptionAdminClient.Pull(pullCtx, req)
	if err != nil {
		return nil, err
	}

	if len(resp.ReceivedMessages) == 0 {
		return nil, nil
	}

	receivedMsg := resp.ReceivedMessages[0]
	return &Message{
		Data:       receivedMsg.Message.Data,
		Attributes: receivedMsg.Message.Attributes,
		AckID:      receivedMsg.AckId,
	}, nil
}

// Publish publishes a message to the topic
func (b *PubSubBroker) Publish(ctx context.Context, message *Message) error {
	result := b.publisher.Publish(ctx, &pubsub.Message{
		Data:       message.Data,
		Attributes: message.Attributes,
	})
	_, err := result.Get(ctx)
	return err
}

// Acknowledge acknowledges a message
func (b *PubSubBroker) Acknowledge(ctx context.Context, ackID string) error {
	req := &pubsubpb.AcknowledgeRequest{
		Subscription: b.subscription,
		AckIds:       []string{ackID},
	}
	return b.subClient.SubscriptionAdminClient.Acknowledge(ctx, req)
}

// Close cleans up resources
func (b *PubSubBroker) Close() error {
	b.publisher.Stop()
	if b.subClient != nil && b.subClient.SubscriptionAdminClient != nil {
		b.subClient.SubscriptionAdminClient.Close()
	}
	if b.topicClient != b.subClient {
		b.topicClient.Close()
	}
	return b.subClient.Close()
}
