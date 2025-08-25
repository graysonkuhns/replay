package cmd_test

import (
	"fmt"
	"strings"
	"testing"

	"replay/e2e_tests/testhelpers"
)

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

// TestDLRExitsOnConsecutiveErrors verifies that DLR also exits after repeated errors
func TestDLRExitsOnConsecutiveErrors(t *testing.T) {
	t.Parallel()

	// Test with an invalid project to trigger repeated errors
	invalidProject := "invalid-project-67890"
	invalidSubscription := fmt.Sprintf("projects/%s/subscriptions/test-sub", invalidProject)
	validTopic := "projects/test/topics/test-topic"

	args := []string{
		"dlr",
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
}
