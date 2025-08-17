package testhelpers

import (
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/pubsub/v2"
)

// BaseIntegrationTest encapsulates common functionality for integration tests
type BaseIntegrationTest struct {
	*testing.T
	Setup     *TestSetup
	TestRunID string
}

// NewBaseIntegrationTest creates a new BaseIntegrationTest instance
func NewBaseIntegrationTest(t *testing.T, testRunID string) *BaseIntegrationTest {
	t.Helper()
	setup := SetupIntegrationTest(t)
	return &BaseIntegrationTest{
		T:         t,
		Setup:     setup,
		TestRunID: testRunID,
	}
}

// PublishMessages publishes messages to the source topic
func (b *BaseIntegrationTest) PublishMessages(messages []pubsub.Message) error {
	b.Helper()
	// Add testRun attribute to all messages if not already present
	for i := range messages {
		if messages[i].Attributes == nil {
			messages[i].Attributes = make(map[string]string)
		}
		if _, exists := messages[i].Attributes["testRun"]; !exists {
			messages[i].Attributes["testRun"] = b.TestRunID
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
func (b *BaseIntegrationTest) RunDLRCommand(inputs string) (string, error) {
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
func (b *BaseIntegrationTest) RunDLRCommandWithArgs(args []string, inputs string) (string, error) {
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
func (b *BaseIntegrationTest) RunMoveCommand(count int) (string, error) {
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
func (b *BaseIntegrationTest) RunMoveCommandWithArgs(args []string) (string, error) {
	b.Helper()
	return RunCLICommand(args)
}

// VerifyMessagesInDestination polls and verifies messages in destination subscription
func (b *BaseIntegrationTest) VerifyMessagesInDestination(expected int) error {
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
func (b *BaseIntegrationTest) VerifyMessagesInSource(expected int) error {
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
func (b *BaseIntegrationTest) GetMessagesFromDestination(expected int) ([]*pubsub.Message, error) {
	b.Helper()
	return PollMessages(
		b.Setup.Context,
		b.Setup.Client,
		b.Setup.GetDestSubscriptionName(),
		b.TestRunID,
		expected,
	)
}

// GetMessagesFromSource retrieves messages from source subscription
func (b *BaseIntegrationTest) GetMessagesFromSource(expected int) ([]*pubsub.Message, error) {
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
func (b *BaseIntegrationTest) WaitForMessagePropagation() {
	b.Helper()
	time.Sleep(30 * time.Second)
}

// CreateTestMessages creates standard test messages with the test run ID
func (b *BaseIntegrationTest) CreateTestMessages(count int, prefix string) []pubsub.Message {
	b.Helper()
	var messages []pubsub.Message
	for i := 1; i <= count; i++ {
		messages = append(messages, pubsub.Message{
			Data: []byte(fmt.Sprintf("%s %d", prefix, i)),
			Attributes: map[string]string{
				"testRun": b.TestRunID,
			},
		})
	}
	return messages
}

// PublishAndWait publishes messages and waits for propagation
func (b *BaseIntegrationTest) PublishAndWait(messages []pubsub.Message) error {
	b.Helper()
	if err := b.PublishMessages(messages); err != nil {
		return err
	}
	b.WaitForMessagePropagation()
	return nil
}
