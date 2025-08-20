package testhelpers

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"replay/constants"

	"cloud.google.com/go/pubsub/v2"
)

// BaseE2ETest encapsulates common functionality for e2e tests
type BaseE2ETest struct {
	*testing.T
	Setup       *TestSetup
	TestRunID   string
	TestContext *TestContext
}

// NewBaseE2ETest creates a new BaseE2ETest instance
func NewBaseE2ETest(t *testing.T, testPrefix string) *BaseE2ETest {
	t.Helper()
	// Generate a unique test run ID with the provided prefix
	testRunID := fmt.Sprintf("%s_%s", testPrefix, GenerateTestRunID())
	setup := SetupE2ETestWithContext(t, testRunID)
	return &BaseE2ETest{
		T:           t,
		Setup:       setup,
		TestRunID:   testRunID,
		TestContext: setup.TestContext,
	}
}

// PublishMessages publishes messages to the source topic
func (b *BaseE2ETest) PublishMessages(messages []pubsub.Message) error {
	b.Helper()
	// Add test context attributes to all messages
	for i := range messages {
		if messages[i].Attributes == nil {
			messages[i].Attributes = make(map[string]string)
		}
		// Add all test context attributes for comprehensive filtering
		for k, v := range b.TestContext.GetAllAttributes() {
			if _, exists := messages[i].Attributes[k]; !exists {
				messages[i].Attributes[k] = v
			}
		}
	}

	_, err := PublishTestMessages(
		b.Setup.Context,
		b.Setup.Client,
		b.Setup.GetSourceTopicName(),
		messages,
		"test-ordering-key",
	)
	return err
}

// RunDLRCommand runs the DLR command with the given inputs
func (b *BaseE2ETest) RunDLRCommand(inputs string) (string, error) {
	b.Helper()
	dlrArgs := []string{
		"dlr",
		"--source-type", constants.BrokerTypeGCPPubSubSubscription,
		"--destination-type", constants.BrokerTypeGCPPubSubTopic,
		"--source", b.Setup.GetSourceSubscriptionName(),
		"--destination", b.Setup.GetDestTopicName(),
	}

	// Create stdin simulator
	simulator, err := NewStdinSimulator(inputs)
	if err != nil {
		return "", fmt.Errorf("failed to create stdin simulator: %w", err)
	}
	defer simulator.Cleanup()

	return RunCLICommand(dlrArgs)
}

// RunDLRCommandWithArgs runs the DLR command with custom arguments
func (b *BaseE2ETest) RunDLRCommandWithArgs(args []string, inputs string) (string, error) {
	b.Helper()
	// Create stdin simulator if inputs provided
	if inputs != "" {
		simulator, err := NewStdinSimulator(inputs)
		if err != nil {
			return "", fmt.Errorf("failed to create stdin simulator: %w", err)
		}
		defer simulator.Cleanup()
	}

	return RunCLICommand(args)
}

// RunMoveCommand runs the move command with an optional count
func (b *BaseE2ETest) RunMoveCommand(count int) (string, error) {
	b.Helper()
	moveArgs := []string{
		"move",
		"--source-type", constants.BrokerTypeGCPPubSubSubscription,
		"--destination-type", constants.BrokerTypeGCPPubSubTopic,
		"--source", b.Setup.GetSourceSubscriptionName(),
		"--destination", b.Setup.GetDestTopicName(),
	}

	if count > 0 {
		moveArgs = append(moveArgs, "--count", fmt.Sprintf("%d", count))
	}

	return RunCLICommand(moveArgs)
}

// RunMoveCommandWithArgs runs the move command with custom arguments
func (b *BaseE2ETest) RunMoveCommandWithArgs(args []string) (string, error) {
	b.Helper()
	return RunCLICommand(args)
}

// VerifyMessagesInDestination polls and verifies messages in destination subscription
func (b *BaseE2ETest) VerifyMessagesInDestination(expected int) error {
	b.Helper()

	ctx := b.Setup.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// Use WaitForMessagesInSubscription with custom options
	opts := &PollingOptions{
		InitialInterval: constants.TestRetryDelay / 2,
		MaxInterval:     constants.TestRetryDelay,
		Multiplier:      1.5,
		MaxElapsedTime:  time.Duration(constants.TestMaxRetries) * constants.TestRetryDelay,
		ProgressCallback: func(elapsed time.Duration, attempt int) {
			b.Logf("Waiting for %d messages in destination (elapsed: %v, attempts: %d)",
				expected, elapsed, attempt)
		},
	}

	return WaitForMessagesInSubscription(ctx, b, b.Setup.GetDestSubscriptionName(), expected, opts)
}

// VerifyMessagesInSource polls and verifies messages in source subscription
func (b *BaseE2ETest) VerifyMessagesInSource(expected int) error {
	b.Helper()

	ctx := b.Setup.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// Use WaitForMessagesInSubscription with custom options
	opts := &PollingOptions{
		InitialInterval: constants.TestRetryDelay / 2,
		MaxInterval:     constants.TestRetryDelay,
		Multiplier:      1.5,
		MaxElapsedTime:  time.Duration(constants.TestMaxRetries) * constants.TestRetryDelay,
		ProgressCallback: func(elapsed time.Duration, attempt int) {
			b.Logf("Waiting for %d messages in source (elapsed: %v, attempts: %d)",
				expected, elapsed, attempt)
		},
	}

	return WaitForMessagesInSubscription(ctx, b, b.Setup.GetSourceSubscriptionName(), expected, opts)
}

// GetMessagesFromDestination retrieves messages from destination subscription
func (b *BaseE2ETest) GetMessagesFromDestination(expected int) ([]*pubsub.Message, error) {
	b.Helper()

	ctx := b.Setup.Context
	if ctx == nil {
		ctx = context.Background()
	}

	var messages []*pubsub.Message

	// Use WaitForCondition to retry with smart polling
	opts := &PollingOptions{
		InitialInterval: constants.TestRetryDelay / 2,
		MaxInterval:     constants.TestRetryDelay,
		Multiplier:      1.5,
		MaxElapsedTime:  time.Duration(constants.TestMaxRetries) * constants.TestRetryDelay,
		ProgressCallback: func(elapsed time.Duration, attempt int) {
			b.Logf("Retrieving %d messages from destination (elapsed: %v, attempts: %d)",
				expected, elapsed, attempt)
		},
	}

	condition := func() (bool, error) {
		received, err := PollMessages(
			b.Setup.Context,
			b.Setup.Client,
			b.Setup.GetDestSubscriptionName(),
			b.TestRunID,
			expected,
		)
		if err == nil {
			messages = received
			return true, nil
		}
		return false, err
	}

	err := WaitForCondition(ctx, fmt.Sprintf("retrieving %d messages from destination", expected), condition, opts)
	if err != nil {
		return nil, err
	}

	return messages, nil
}

// GetMessagesFromSource retrieves messages from source subscription
func (b *BaseE2ETest) GetMessagesFromSource(expected int) ([]*pubsub.Message, error) {
	b.Helper()
	return PollMessages(
		b.Setup.Context,
		b.Setup.Client,
		b.Setup.GetSourceSubscriptionName(),
		b.TestRunID,
		expected,
	)
}

// WaitForMessagePropagation waits for messages to propagate through PubSub
func (b *BaseE2ETest) WaitForMessagePropagation() {
	b.Helper()
	ctx := b.Setup.Context
	if ctx == nil {
		ctx = context.Background()
	}
	if err := WaitWithBackoff(ctx, "message propagation", constants.TestMessagePropagation, b); err != nil {
		// Log the error but don't fail the test since this is just a wait
		b.Logf("Warning during message propagation wait: %v", err)
	}
}

// CreateTestMessages creates standard test messages with test context attributes
func (b *BaseE2ETest) CreateTestMessages(count int, prefix string) []pubsub.Message {
	b.Helper()
	var messages []pubsub.Message
	for i := 1; i <= count; i++ {
		messages = append(messages, pubsub.Message{
			Data:       []byte(fmt.Sprintf("%s %d", prefix, i)),
			Attributes: b.TestContext.GetAllAttributes(),
		})
	}
	return messages
}

// PublishAndWait publishes messages and waits for propagation
func (b *BaseE2ETest) PublishAndWait(messages []pubsub.Message) error {
	b.Helper()
	if err := b.PublishMessages(messages); err != nil {
		return err
	}
	b.WaitForMessagePropagation()
	return nil
}

// CreateTempFile creates a temporary file with automatic tracking and cleanup
func (b *BaseE2ETest) CreateTempFile(pattern string) (*os.File, error) {
	b.Helper()
	file, err := os.CreateTemp("", pattern)
	if err != nil {
		return nil, err
	}

	// Track the file for cleanup
	b.TestContext.TrackTempFile(file.Name())

	// Register cleanup
	b.Cleanup(func() {
		file.Close()
		if err := os.Remove(file.Name()); err == nil {
			// Untrack the file after successful cleanup
			b.TestContext.UntrackTempFile(file.Name())
		}
	})

	return file, nil
}
