package testhelpers

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/pubsub/v2"
	pubsubpb "cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"google.golang.org/protobuf/types/known/durationpb"
)

// TestSetup holds the common setup for e2e tests
type TestSetup struct {
	Context         context.Context
	Client          *pubsub.Client
	ProjectID       string
	SourceTopicName string
	SourceSubName   string
	DestTopicName   string
	DestSubName     string
	// Resource names (full paths) for use with v2 API
	SourceSubFullName   string
	DestSubFullName     string
	SourceTopicFullName string
	DestTopicFullName   string
	// Test context for isolation and tracking
	TestContext *TestContext
}

// GenerateTestRunID generates a unique test run ID
func GenerateTestRunID() string {
	return fmt.Sprintf("test-run-%d-%d", time.Now().UnixNano(), rand.Intn(10000))
}

// SetupE2ETest creates a common setup for e2e tests
func SetupE2ETest(t *testing.T) *TestSetup {
	return SetupE2ETestWithContext(t, GenerateTestRunID())
}

// SetupE2ETestWithContext creates a common setup with a specific test context
func SetupE2ETestWithContext(t *testing.T, testRunID string) *TestSetup {
	ctx := context.Background()
	projectID := os.Getenv("GCP_PROJECT")
	if projectID == "" {
		t.Fatal("GCP_PROJECT environment variable must be set")
	}

	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		t.Fatalf("Failed to create PubSub client: %v", err)
	}

	// Create test context for isolation
	testCtx := NewTestContext(t, testRunID)

	// Generate unique resource names using test context
	sourceTopicName := testCtx.GenerateResourceName("topic", "events_dead_letter")
	sourceSubName := testCtx.GenerateResourceName("sub", "events_dead_letter")
	destTopicName := testCtx.GenerateResourceName("topic", "events")
	destSubName := testCtx.GenerateResourceName("sub", "events")

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
		Name:                     sourceSubFullName,
		Topic:                    sourceTopicFullName,
		AckDeadlineSeconds:       60,
		EnableMessageOrdering:    true,
		MessageRetentionDuration: durationpb.New(604800 * time.Second), // 7 days default
	})
	if err != nil {
		t.Fatalf("Failed to create source subscription %s: %v", sourceSubFullName, err)
	}

	_, err = subAdmin.CreateSubscription(ctx, &pubsubpb.Subscription{
		Name:                     destSubFullName,
		Topic:                    destTopicFullName,
		AckDeadlineSeconds:       60,
		EnableMessageOrdering:    true,
		MessageRetentionDuration: durationpb.New(604800 * time.Second), // 7 days default
	})
	if err != nil {
		t.Fatalf("Failed to create destination subscription %s: %v", destSubFullName, err)
	}

	// Track resources in test context
	testCtx.TrackTopic(sourceTopicFullName)
	testCtx.TrackTopic(destTopicFullName)
	testCtx.TrackSubscription(sourceSubFullName)
	testCtx.TrackSubscription(destSubFullName)

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
		TestContext:         testCtx,
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
		} else {
			// Untrack successfully deleted resource
			testCtx.UntrackSubscription(sourceSubFullName)
		}

		if err := subAdmin.DeleteSubscription(ctx, &pubsubpb.DeleteSubscriptionRequest{
			Subscription: destSubFullName,
		}); err != nil {
			log.Printf("Failed to delete destination subscription %s: %v", destSubFullName, err)
		} else {
			// Untrack successfully deleted resource
			testCtx.UntrackSubscription(destSubFullName)
		}

		// Delete topics using v2 admin client
		topicAdmin := client.TopicAdminClient
		if err := topicAdmin.DeleteTopic(ctx, &pubsubpb.DeleteTopicRequest{
			Topic: sourceTopicFullName,
		}); err != nil {
			log.Printf("Failed to delete source topic %s: %v", sourceTopicFullName, err)
		} else {
			// Untrack successfully deleted resource
			testCtx.UntrackTopic(sourceTopicFullName)
		}

		if err := topicAdmin.DeleteTopic(ctx, &pubsubpb.DeleteTopicRequest{
			Topic: destTopicFullName,
		}); err != nil {
			log.Printf("Failed to delete destination topic %s: %v", destTopicFullName, err)
		} else {
			// Untrack successfully deleted resource
			testCtx.UntrackTopic(destTopicFullName)
		}

		client.Close()

		// Check for resource leaks
		snapshot := testCtx.GetTrackedResources()
		if snapshot.HasLeaks() {
			t.Errorf("Resource leak detected in test %s:\n%s", t.Name(), snapshot.GetLeakSummary())
		}
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
