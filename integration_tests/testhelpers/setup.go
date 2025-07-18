package testhelpers

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
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
	SourceTopic     *pubsub.Topic
	DestTopic       *pubsub.Topic
}

// generateUniqueResourceName creates a unique resource name for test isolation
func generateUniqueResourceName(baseName string, testName string) string {
	// Clean test name to make it suitable for resource names
	cleanTestName := strings.ReplaceAll(testName, "/", "_")
	cleanTestName = strings.ReplaceAll(cleanTestName, " ", "_")
	cleanTestName = strings.ToLower(cleanTestName)

	// Add timestamp and random suffix for uniqueness
	timestamp := time.Now().Unix()
	randSuffix := rand.Intn(1000)

	return fmt.Sprintf("%s_%s_%d_%d", baseName, cleanTestName, timestamp, randSuffix)
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

	// Generate unique resource names for this test
	testName := t.Name()
	sourceTopicName := generateUniqueResourceName("test-events-dead-letter", testName)
	sourceSubName := generateUniqueResourceName("test-events-dead-letter-subscription", testName)
	destTopicName := generateUniqueResourceName("test-events", testName)
	destSubName := generateUniqueResourceName("test-events-subscription", testName)

	// Create topics
	sourceTopic, err := client.CreateTopic(ctx, sourceTopicName)
	if err != nil {
		t.Fatalf("Failed to create source topic %s: %v", sourceTopicName, err)
	}
	log.Printf("Created source topic: %s", sourceTopicName)

	destTopic, err := client.CreateTopic(ctx, destTopicName)
	if err != nil {
		t.Fatalf("Failed to create destination topic %s: %v", destTopicName, err)
	}
	log.Printf("Created destination topic: %s", destTopicName)

	// Create subscriptions
	sourceSub, err := client.CreateSubscription(ctx, sourceSubName, pubsub.SubscriptionConfig{
		Topic:       sourceTopic,
		AckDeadline: 60 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create source subscription %s: %v", sourceSubName, err)
	}
	log.Printf("Created source subscription: %s", sourceSubName)

	destSub, err := client.CreateSubscription(ctx, destSubName, pubsub.SubscriptionConfig{
		Topic:       destTopic,
		AckDeadline: 60 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create destination subscription %s: %v", destSubName, err)
	}
	log.Printf("Created destination subscription: %s", destSubName)

	setup := &TestSetup{
		Context:         ctx,
		Client:          client,
		ProjectID:       projectID,
		SourceTopicName: sourceTopicName,
		SourceSubName:   sourceSubName,
		DestTopicName:   destTopicName,
		DestSubName:     destSubName,
		SourceSub:       sourceSub,
		DestSub:         destSub,
		SourceTopic:     sourceTopic,
		DestTopic:       destTopic,
	}

	// Setup cleanup to delete resources after test
	t.Cleanup(func() {
		log.Printf("Cleaning up resources for test: %s", testName)

		// Delete subscriptions first
		if err := sourceSub.Delete(ctx); err != nil {
			log.Printf("Failed to delete source subscription %s: %v", sourceSubName, err)
		} else {
			log.Printf("Deleted source subscription: %s", sourceSubName)
		}

		if err := destSub.Delete(ctx); err != nil {
			log.Printf("Failed to delete destination subscription %s: %v", destSubName, err)
		} else {
			log.Printf("Deleted destination subscription: %s", destSubName)
		}

		// Delete topics
		if err := sourceTopic.Delete(ctx); err != nil {
			log.Printf("Failed to delete source topic %s: %v", sourceTopicName, err)
		} else {
			log.Printf("Deleted source topic: %s", sourceTopicName)
		}

		if err := destTopic.Delete(ctx); err != nil {
			log.Printf("Failed to delete destination topic %s: %v", destTopicName, err)
		} else {
			log.Printf("Deleted destination topic: %s", destTopicName)
		}

		client.Close()
	})

	return setup
}

// PurgeSubscriptions purges both source and destination subscriptions
// Deprecated: This method is no longer needed since tests now create unique resources
// that don't share topics and subscriptions between tests. Kept for backward compatibility.
func (s *TestSetup) PurgeSubscriptions(t *testing.T) {
	log.Printf("Purging subscriptions (deprecated - not needed with unique resources)")
	// Purge source subscription
	log.Printf("Purging source subscription: %s", s.SourceSubName)
	if err := PurgeSubscription(s.Context, s.SourceSub); err != nil {
		t.Fatalf("Failed to purge source subscription: %v", err)
	}

	// Purge destination subscription
	log.Printf("Purging destination subscription: %s", s.DestSubName)
	if err := PurgeSubscription(s.Context, s.DestSub); err != nil {
		t.Fatalf("Failed to purge destination subscription: %v", err)
	}
}

// PurgeSourceSubscription purges only the source subscription
// Deprecated: This method is no longer needed since tests now create unique resources
// that don't share topics and subscriptions between tests. Kept for backward compatibility.
func (s *TestSetup) PurgeSourceSubscription(t *testing.T) {
	log.Printf("Purging source subscription: %s (deprecated - not needed with unique resources)", s.SourceSubName)
	if err := PurgeSubscription(s.Context, s.SourceSub); err != nil {
		t.Fatalf("Failed to purge source subscription: %v", err)
	}
}

// GetSourceTopic returns the source topic
func (s *TestSetup) GetSourceTopic() *pubsub.Topic {
	return s.SourceTopic
}

// GetDestTopic returns the destination topic
func (s *TestSetup) GetDestTopic() *pubsub.Topic {
	return s.DestTopic
}
