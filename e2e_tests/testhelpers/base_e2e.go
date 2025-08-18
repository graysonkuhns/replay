package testhelpers

import (
	"fmt"
	"os"
	"testing"
	"time"

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
		"--source-type", "GCP_PUBSUB_SUBSCRIPTION",
		"--destination-type", "GCP_PUBSUB_TOPIC",
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
		"--source-type", "GCP_PUBSUB_SUBSCRIPTION",
		"--destination-type", "GCP_PUBSUB_TOPIC",
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
	received, err := PollMessages(
		b.Setup.Context,
		b.Setup.Client,
		b.Setup.GetDestSubscriptionName(),
		b.TestRunID,
		expected,
	)
	if err != nil {
		return fmt.Errorf("error polling destination subscription: %w", err)
	}

	if len(received) != expected {
		return fmt.Errorf("expected %d messages in destination, got %d", expected, len(received))
	}

	return nil
}

// VerifyMessagesInSource polls and verifies messages in source subscription
func (b *BaseE2ETest) VerifyMessagesInSource(expected int) error {
	b.Helper()
	received, err := PollMessages(
		b.Setup.Context,
		b.Setup.Client,
		b.Setup.GetSourceSubscriptionName(),
		b.TestRunID,
		expected,
	)
	if err != nil {
		return fmt.Errorf("error polling source subscription: %w", err)
	}

	if len(received) != expected {
		return fmt.Errorf("expected %d messages in source, got %d", expected, len(received))
	}

	return nil
}

// GetMessagesFromDestination retrieves messages from destination subscription
func (b *BaseE2ETest) GetMessagesFromDestination(expected int) ([]*pubsub.Message, error) {
	b.Helper()
	// Retry mechanism for improved reliability in nightly tests
	const maxRetries = 3
	const retryDelay = 5 * time.Second

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		received, err := PollMessages(
			b.Setup.Context,
			b.Setup.Client,
			b.Setup.GetDestSubscriptionName(),
			b.TestRunID,
			expected,
		)

		if err == nil {
			return received, nil
		}

		lastErr = err
		if attempt < maxRetries {
			b.Logf("Attempt %d failed to get %d messages from destination: %v. Retrying in %v...",
				attempt, expected, err, retryDelay)
			time.Sleep(retryDelay)
		}
	}

	return nil, fmt.Errorf("failed to get messages after %d attempts: %w", maxRetries, lastErr)
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
	time.Sleep(30 * time.Second)
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
