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

	"cloud.google.com/go/pubsub/v2"
	pubsubpb "cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"google.golang.org/protobuf/types/known/durationpb"
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
	// Resource names (full paths) for use with v2 API
	SourceSubFullName  string
	DestSubFullName    string
	SourceTopicFullName string
	DestTopicFullName   string
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

	// Create full resource paths
	sourceTopicFullName := fmt.Sprintf("projects/%s/topics/%s", projectID, sourceTopicName)
	sourceSubFullName := fmt.Sprintf("projects/%s/subscriptions/%s", projectID, sourceSubName)
	destTopicFullName := fmt.Sprintf("projects/%s/topics/%s", projectID, destTopicName)
	destSubFullName := fmt.Sprintf("projects/%s/subscriptions/%s", projectID, destSubName)

	// Create topics using v2 admin client
	topicAdmin := client.TopicAdminClient
	_, err = topicAdmin.CreateTopic(ctx, &pubsubpb.Topic{
		Name: sourceTopicFullName,
	})
	if err != nil {
		t.Fatalf("Failed to create source topic %s: %v", sourceTopicFullName, err)
	}
	// Suppress creation logs to avoid interfering with parallel test output

	_, err = topicAdmin.CreateTopic(ctx, &pubsubpb.Topic{
		Name: destTopicFullName,
	})
	if err != nil {
		t.Fatalf("Failed to create destination topic %s: %v", destTopicFullName, err)
	}

	// Create subscriptions using v2 admin client
	subAdmin := client.SubscriptionAdminClient
	_, err = subAdmin.CreateSubscription(ctx, &pubsubpb.Subscription{
		Name:                       sourceSubFullName,
		Topic:                      sourceTopicFullName,
		AckDeadlineSeconds:         60,
		EnableMessageOrdering:      true,
		MessageRetentionDuration:   durationpb.New(604800 * time.Second), // 7 days default
	})
	if err != nil {
		t.Fatalf("Failed to create source subscription %s: %v", sourceSubFullName, err)
	}

	_, err = subAdmin.CreateSubscription(ctx, &pubsubpb.Subscription{
		Name:                       destSubFullName,
		Topic:                      destTopicFullName,
		AckDeadlineSeconds:         60,
		EnableMessageOrdering:      true,
		MessageRetentionDuration:   durationpb.New(604800 * time.Second), // 7 days default
	})
	if err != nil {
		t.Fatalf("Failed to create destination subscription %s: %v", destSubFullName, err)
	}

	setup := &TestSetup{
		Context:             ctx,
		Client:              client,
		ProjectID:           projectID,
		SourceTopicName:     sourceTopicName,
		SourceSubName:       sourceSubName,
		DestTopicName:       destTopicName,
		DestSubName:         destSubName,
		SourceSubFullName:   sourceSubFullName,
		DestSubFullName:     destSubFullName,
		SourceTopicFullName: sourceTopicFullName,
		DestTopicFullName:   destTopicFullName,
	}

	// Setup cleanup to delete resources after test
	t.Cleanup(func() {
		// Suppress cleanup logs to avoid interfering with parallel test output
		// Only log errors as they are important for debugging

		// Delete subscriptions first using v2 admin client
		subAdmin := client.SubscriptionAdminClient
		if err := subAdmin.DeleteSubscription(ctx, &pubsubpb.DeleteSubscriptionRequest{
			Subscription: sourceSubFullName,
		}); err != nil {
			log.Printf("Failed to delete source subscription %s: %v", sourceSubFullName, err)
		}

		if err := subAdmin.DeleteSubscription(ctx, &pubsubpb.DeleteSubscriptionRequest{
			Subscription: destSubFullName,
		}); err != nil {
			log.Printf("Failed to delete destination subscription %s: %v", destSubFullName, err)
		}

		// Delete topics using v2 admin client
		topicAdmin := client.TopicAdminClient
		if err := topicAdmin.DeleteTopic(ctx, &pubsubpb.DeleteTopicRequest{
			Topic: sourceTopicFullName,
		}); err != nil {
			log.Printf("Failed to delete source topic %s: %v", sourceTopicFullName, err)
		}

		if err := topicAdmin.DeleteTopic(ctx, &pubsubpb.DeleteTopicRequest{
			Topic: destTopicFullName,
		}); err != nil {
			log.Printf("Failed to delete destination topic %s: %v", destTopicFullName, err)
		}

		client.Close()
	})

	return setup
}

// GetSourceTopicName returns the source topic full resource name
func (s *TestSetup) GetSourceTopicName() string {
	return s.SourceTopicFullName
}

// GetDestTopicName returns the destination topic full resource name
func (s *TestSetup) GetDestTopicName() string {
	return s.DestTopicFullName
}

// GetSourceSubscriptionName returns the source subscription full resource name
func (s *TestSetup) GetSourceSubscriptionName() string {
	return s.SourceSubFullName
}

// GetDestSubscriptionName returns the destination subscription full resource name
func (s *TestSetup) GetDestSubscriptionName() string {
	return s.DestSubFullName
}
