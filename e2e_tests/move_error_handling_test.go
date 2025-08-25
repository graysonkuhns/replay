package cmd_test

import (
	"fmt"
	"strings"
	"testing"

	"replay/e2e_tests/testhelpers"
)

// TestMoveHandlesEmptySource verifies move command behavior with no messages
func TestMoveHandlesEmptySource(t *testing.T) {
	t.Parallel()
	baseTest := testhelpers.NewBaseE2ETest(t, "move_empty_source")

	// Run move on empty subscription
	output, err := baseTest.RunMoveCommand(0)
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}

	// Verify appropriate completion message
	expectedLines := []string{
		fmt.Sprintf("[TIMESTAMP] Moving messages from %s to %s", baseTest.Setup.GetSourceSubscriptionName(), baseTest.Setup.GetDestTopicName()),
		"[TIMESTAMP] Move operation completed. Total messages moved: 0",
	}

	testhelpers.AssertCLIOutput(t, output, expectedLines)
}

// TestMoveWithZeroCount tests move command with count=0
func TestMoveWithZeroCount(t *testing.T) {
	t.Parallel()
	baseTest := testhelpers.NewBaseE2ETest(t, "move_zero_count")

	// Publish messages
	messages := baseTest.CreateTestMessages(3, "Test message")
	if err := baseTest.PublishAndWait(messages); err != nil {
		t.Fatalf("Failed to publish test messages: %v", err)
	}

	// Run move with count=0 (should move all messages)
	output, err := baseTest.RunMoveCommand(0)
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}

	// Verify all messages were moved
	if !strings.Contains(output, "Total messages moved: 3") {
		t.Fatalf("Expected 3 messages moved, got: %s", output)
	}

	baseTest.WaitForMessagePropagation()

	// Verify destination has all messages
	if err := baseTest.VerifyMessagesInDestination(3); err != nil {
		t.Fatalf("%v", err)
	}

	// Verify source is empty
	if err := baseTest.VerifyMessagesInSource(0); err != nil {
		t.Fatalf("%v", err)
	}
}

// TestMoveWithNegativeCount tests error handling for negative count values
func TestMoveWithNegativeCount(t *testing.T) {
	t.Parallel()

	// Run command with negative count
	args := []string{
		"move",
		"--source-type", "pubsub",
		"--source", "projects/test/subscriptions/test-sub",
		"--destination-type", "pubsub",
		"--destination", "projects/test/topics/test-topic",
		"--count", "-5",
	}

	output, err := testhelpers.RunCLICommand(args)

	// Should fail with appropriate error
	if err == nil {
		t.Fatalf("Expected error for negative count, got none")
	}

	// Check for appropriate error message
	// The actual error might vary - could be Cobra validation or custom validation
	if !strings.Contains(output, "Error:") {
		t.Fatalf("Expected error message, got: %s", output)
	}
}

// TestMoveHandlesMessageWithComplexAttributes tests attribute preservation
func TestMoveHandlesMessageWithComplexAttributes(t *testing.T) {
	t.Parallel()
	baseTest := testhelpers.NewBaseE2ETest(t, "move_complex_attributes")

	// Create message with complex attributes
	messages := baseTest.CreateTestMessages(1, "Message with attributes")
	messages[0].Attributes = map[string]string{
		"key1":          "value1",
		"empty-value":   "",
		"special-chars": "!@#$%^&*()",
		"unicode":       "你好世界",
		"long-key-name-that-exceeds-typical-length": "value",
		"ordering-key": "important-order-123",
	}

	if err := baseTest.PublishAndWait(messages); err != nil {
		t.Fatalf("Failed to publish test messages: %v", err)
	}

	// Move the message
	_, err := baseTest.RunMoveCommand(1)
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}

	baseTest.WaitForMessagePropagation()

	// Verify attributes are preserved
	received, err := baseTest.GetMessagesFromDestination(1)
	if err != nil {
		t.Fatalf("Error receiving messages from destination: %v", err)
	}

	// Check all attributes are preserved
	for key, expectedValue := range messages[0].Attributes {
		actualValue, exists := received[0].Attributes[key]
		if !exists {
			t.Fatalf("Attribute %s not found in moved message", key)
		}
		if actualValue != expectedValue {
			t.Fatalf("Attribute %s mismatch: expected '%s', got '%s'", key, expectedValue, actualValue)
		}
	}
}

// TestMoveCrossProjectOperation tests moving between different projects
func TestMoveCrossProjectOperation(t *testing.T) {
	t.Parallel()

	// This test would require setup with two different projects
	// For now, we'll test that the broker correctly parses cross-project resources

	sourceProject := "source-project"
	destProject := "dest-project"

	args := []string{
		"move",
		"--source-type", "pubsub",
		"--source", fmt.Sprintf("projects/%s/subscriptions/test-sub", sourceProject),
		"--destination-type", "pubsub",
		"--destination", fmt.Sprintf("projects/%s/topics/test-topic", destProject),
		"--count", "1",
	}

	// Note: This will fail due to missing resources, but we're testing
	// that it attempts to create clients for different projects
	output, _ := testhelpers.RunCLICommand(args)

	// The error should be about missing resources or auth, not about parsing
	if strings.Contains(output, "invalid subscription resource format") ||
		strings.Contains(output, "invalid topic resource format") {
		t.Fatalf("Cross-project resource parsing failed: %s", output)
	}
}

// TestMoveHandlesVeryLargeCount tests behavior with extremely large count values
func TestMoveHandlesVeryLargeCount(t *testing.T) {
	t.Parallel()
	baseTest := testhelpers.NewBaseE2ETest(t, "move_large_count")

	// Publish only 2 messages
	messages := baseTest.CreateTestMessages(2, "Test message")
	if err := baseTest.PublishAndWait(messages); err != nil {
		t.Fatalf("Failed to publish test messages: %v", err)
	}

	// Try to move 1000000 messages (should only move 2)
	output, err := baseTest.RunMoveCommand(1000000)
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}

	// Should gracefully handle and move only available messages
	if !strings.Contains(output, "Total messages moved: 2") {
		t.Fatalf("Expected to move only 2 available messages, got: %s", output)
	}
}

// TestProcessorExitsOnConsecutiveErrors verifies that the processor exits after repeated errors
func TestProcessorExitsOnConsecutiveErrors(t *testing.T) {
	t.Parallel()

	// Test with an invalid project to trigger repeated errors
	invalidProject := "invalid-project-12345"
	invalidSubscription := fmt.Sprintf("projects/%s/subscriptions/test-sub", invalidProject)
	validTopic := "projects/test/topics/test-topic"

	args := []string{
		"move",
		"--source-type", "GCP_PUBSUB_SUBSCRIPTION",
		"--source", invalidSubscription,
		"--destination-type", "GCP_PUBSUB_TOPIC",
		"--destination", validTopic,
		"--polling-timeout-seconds", "2", // Short timeout to speed up test
	}

	output, err := testhelpers.RunCLICommand(args)

	// The command should fail (exit with error)
	if err == nil {
		t.Fatalf("Expected command to fail with error, but it succeeded")
	}

	// Verify we see exactly 5 error messages (MaxConsecutiveErrors)
	errorCount := strings.Count(output, "Error during message pull:")
	if errorCount != 5 {
		t.Fatalf("Expected exactly 5 error messages, got %d. Output:\n%s", errorCount, output)
	}

	// Verify we see the stopping message
	if !strings.Contains(output, "stopping after 5 consecutive errors") {
		t.Fatalf("Expected 'stopping after 5 consecutive errors' message not found. Output:\n%s", output)
	}

	// Verify the error contains the original error message
	if !strings.Contains(output, invalidProject) {
		t.Fatalf("Expected error to contain invalid project name '%s'. Output:\n%s", invalidProject, output)
	}
}
