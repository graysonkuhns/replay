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
	// Suppress creation logs to avoid interfering with parallel test output

	destTopic, err := client.CreateTopic(ctx, destTopicName)
	if err != nil {
		t.Fatalf("Failed to create destination topic %s: %v", destTopicName, err)
	}

	// Create subscriptions
	sourceSub, err := client.CreateSubscription(ctx, sourceSubName, pubsub.SubscriptionConfig{
		Topic:       sourceTopic,
		AckDeadline: 60 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create source subscription %s: %v", sourceSubName, err)
	}

	destSub, err := client.CreateSubscription(ctx, destSubName, pubsub.SubscriptionConfig{
		Topic:       destTopic,
		AckDeadline: 60 * time.Second,
	})
	if err != nil {
		t.Fatalf("Failed to create destination subscription %s: %v", destSubName, err)
	}

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
		// Suppress cleanup logs to avoid interfering with parallel test output
		// Only log errors as they are important for debugging

		// Delete subscriptions first
		if err := sourceSub.Delete(ctx); err != nil {
			log.Printf("Failed to delete source subscription %s: %v", sourceSubName, err)
		}

		if err := destSub.Delete(ctx); err != nil {
			log.Printf("Failed to delete destination subscription %s: %v", destSubName, err)
		}

		// Delete topics
		if err := sourceTopic.Delete(ctx); err != nil {
			log.Printf("Failed to delete source topic %s: %v", sourceTopicName, err)
		}

		if err := destTopic.Delete(ctx); err != nil {
			log.Printf("Failed to delete destination topic %s: %v", destTopicName, err)
		}

		client.Close()
	})

	return setup
}

// GetSourceTopic returns the source topic
func (s *TestSetup) GetSourceTopic() *pubsub.Topic {
	return s.SourceTopic
}

// GetDestTopic returns the destination topic
func (s *TestSetup) GetDestTopic() *pubsub.Topic {
	return s.DestTopic
}
