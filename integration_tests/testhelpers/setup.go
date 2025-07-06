package testhelpers

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/pubsub"
)

// TestSetup holds the common setup for integration tests
type TestSetup struct {
	Context         context.Context
	Client          *pubsub.Client
	ProjectID       string
	SourceTopicName string
	SourceSubName   string
	DestTopicName   string
	DestSubName     string
	SourceSub       *pubsub.Subscription
	DestSub         *pubsub.Subscription
}

// SetupIntegrationTest creates a common setup for integration tests
func SetupIntegrationTest(t *testing.T) *TestSetup {
	ctx := context.Background()
	projectID := os.Getenv("GCP_PROJECT")
	if projectID == "" {
		t.Fatal("GCP_PROJECT environment variable must be set")
	}

	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		t.Fatalf("Failed to create PubSub client: %v", err)
	}

	// Standard resource names used across all tests
	sourceTopicName := "default-events-dead-letter"
	sourceSubName := "default-events-dead-letter-subscription"
	destTopicName := "default-events"
	destSubName := "default-events-subscription"

	setup := &TestSetup{
		Context:         ctx,
		Client:          client,
		ProjectID:       projectID,
		SourceTopicName: sourceTopicName,
		SourceSubName:   sourceSubName,
		DestTopicName:   destTopicName,
		DestSubName:     destSubName,
		SourceSub:       client.Subscription(sourceSubName),
		DestSub:         client.Subscription(destSubName),
	}

	// Setup cleanup
	t.Cleanup(func() {
		client.Close()
	})

	return setup
}

// PurgeSubscriptions purges both source and destination subscriptions
func (s *TestSetup) PurgeSubscriptions(t *testing.T) {
	// Purge source subscription
	log.Printf("Purging source subscription: %s", s.SourceSubName)
	if err := PurgeSubscription(s.Context, s.SourceSub); err != nil {
		t.Fatalf("Failed to purge source subscription: %v", err)
	}

	// Purge destination subscription
	log.Printf("Purging destination subscription: %s", s.DestSubName)
	destCtx, destCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer destCancel()
	if err := PurgeSubscription(destCtx, s.DestSub); err != nil {
		t.Fatalf("Failed to purge destination subscription: %v", err)
	}
}

// PurgeSourceSubscription purges only the source subscription
func (s *TestSetup) PurgeSourceSubscription(t *testing.T) {
	log.Printf("Purging source subscription: %s", s.SourceSubName)
	if err := PurgeSubscription(s.Context, s.SourceSub); err != nil {
		t.Fatalf("Failed to purge source subscription: %v", err)
	}
}

// GetSourceTopic returns the source topic
func (s *TestSetup) GetSourceTopic() *pubsub.Topic {
	return s.Client.Topic(s.SourceTopicName)
}

// GetDestTopic returns the destination topic
func (s *TestSetup) GetDestTopic() *pubsub.Topic {
	return s.Client.Topic(s.DestTopicName)
}