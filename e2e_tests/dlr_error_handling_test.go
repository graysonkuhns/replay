package cmd_test

import (
	"fmt"
	"strings"
	"testing"

	"replay/e2e_tests/testhelpers"

	"cloud.google.com/go/pubsub/v2"
)

// TestDLRHandlesEmptySubscription verifies behavior when no messages are available
func TestDLRHandlesEmptySubscription(t *testing.T) {
	t.Parallel()
	baseTest := testhelpers.NewBaseE2ETest(t, "dlr_empty_subscription")

	// Run DLR on empty subscription
	output, err := baseTest.RunDLRCommand("")
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}

	// Verify appropriate message for empty subscription
	expectedLines := []string{
		fmt.Sprintf("Starting DLR review from %s", baseTest.Setup.GetSourceSubscriptionName()),
		"",
		"Dead-lettered messages review completed. Total messages processed: 0",
	}

	testhelpers.AssertCLIOutput(t, output, expectedLines)
}

// TestDLRHandlesMessageWithoutCustomAttributes verifies handling of messages with only test framework attributes
func TestDLRHandlesMessageWithoutCustomAttributes(t *testing.T) {
	t.Parallel()
	baseTest := testhelpers.NewBaseE2ETest(t, "dlr_no_custom_attrs")

	// Create message without custom attributes (will have test framework attributes)
	messages := baseTest.CreateTestMessages(1, "Message without custom attributes")

	if err := baseTest.PublishAndWait(messages); err != nil {
		t.Fatalf("Failed to publish test messages: %v", err)
	}

	// Process message
	output, err := baseTest.RunDLRCommand("m\n")
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}

	// Verify message display and processing
	if !strings.Contains(output, "Message without custom attributes") {
		t.Fatalf("Expected message content not found in output")
	}
	// Verify attributes are displayed (should have test framework attributes)
	if !strings.Contains(output, "Attributes: map[") {
		t.Fatalf("Expected attributes display not found")
	}
	// Verify test framework attributes are present
	if !strings.Contains(output, "testName:") {
		t.Fatalf("Expected test framework attributes not found")
	}
	if !strings.Contains(output, "moved successfully") {
		t.Fatalf("Expected move success message not found")
	}
}

// TestDLRHandlesMessageWithNilAttributes verifies handling of messages with truly nil attributes
func TestDLRHandlesMessageWithNilAttributes(t *testing.T) {
	t.Parallel()
	baseTest := testhelpers.NewBaseE2ETest(t, "dlr_nil_attributes")

	// Publish a message with nil attributes directly (bypassing test framework)
	ctx := baseTest.Setup.Context
	publisher := baseTest.Setup.Client.Publisher(baseTest.Setup.GetSourceTopicName())
	defer publisher.Stop()

	result := publisher.Publish(ctx, &pubsub.Message{
		Data:       []byte("Message with nil attributes"),
		Attributes: nil,
	})
	if _, err := result.Get(ctx); err != nil {
		t.Fatalf("Failed to publish message: %v", err)
	}

	// Wait for message propagation
	baseTest.WaitForMessagePropagation()

	// Process message
	output, err := baseTest.RunDLRCommand("m\n")
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}

	// Verify message display and processing
	if !strings.Contains(output, "Message with nil attributes") {
		t.Fatalf("Expected message content not found in output")
	}
	// When attributes are nil, Go's %v formatter shows "<nil>"
	if !strings.Contains(output, "Attributes: <nil>") && !strings.Contains(output, "Attributes: map[]") {
		t.Fatalf("Expected nil/empty attributes display not found in output:\n%s", output)
	}
	if !strings.Contains(output, "moved successfully") {
		t.Fatalf("Expected move success message not found")
	}
}

// TestDLRHandlesVeryLongMessage tests processing of messages near size limits
func TestDLRHandlesVeryLongMessage(t *testing.T) {
	t.Parallel()
	baseTest := testhelpers.NewBaseE2ETest(t, "dlr_long_message")

	// Create a large message (1MB)
	largeData := strings.Repeat("A", 1024*1024)
	messages := baseTest.CreateTestMessages(1, largeData)

	if err := baseTest.PublishAndWait(messages); err != nil {
		t.Fatalf("Failed to publish test messages: %v", err)
	}

	// Process the large message
	output, err := baseTest.RunDLRCommand("m\n")
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}

	// Verify successful processing
	if !strings.Contains(output, "moved successfully") {
		t.Fatalf("Expected move success message not found")
	}

	// Verify message arrived at destination intact
	baseTest.WaitForMessagePropagation()
	received, err := baseTest.GetMessagesFromDestination(1)
	if err != nil {
		t.Fatalf("Error receiving messages from destination: %v", err)
	}

	if len(received[0].Data) != len(largeData) {
		t.Fatalf("Message size mismatch: expected %d, got %d", len(largeData), len(received[0].Data))
	}
}

// TestDLRInvalidResourceName tests handling of malformed subscription names
func TestDLRInvalidResourceName(t *testing.T) {
	t.Parallel()

	// Test with invalid subscription format
	invalidSub := "invalid-subscription-format"

	// Run command with invalid subscription
	args := []string{
		"dlr",
		"--source", "pubsub",
		"--source-location", invalidSub,
		"--destination", "pubsub",
		"--destination-location", "projects/test/topics/valid-topic",
	}

	output, err := testhelpers.RunCLICommand(args)

	// Should fail with appropriate error
	if err == nil {
		t.Fatalf("Expected error for invalid subscription format, got none")
	}

	if !strings.Contains(output, "invalid subscription resource format") {
		t.Fatalf("Expected specific error message about invalid format, got: %s", output)
	}
}

// TestDLRHandlesSpecialCharactersInInput tests interactive input with special characters
func TestDLRHandlesSpecialCharactersInInput(t *testing.T) {
	t.Parallel()
	baseTest := testhelpers.NewBaseE2ETest(t, "dlr_special_input")

	messages := baseTest.CreateTestMessages(3, "Test message")
	if err := baseTest.PublishAndWait(messages); err != nil {
		t.Fatalf("Failed to publish test messages: %v", err)
	}

	// Test various special inputs
	specialInputs := "!@#$%^&*()\nm\nd\n"
	output, err := baseTest.RunDLRCommand(specialInputs)
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}

	// Verify invalid input handling
	if !strings.Contains(output, "Invalid choice") {
		t.Fatalf("Expected invalid choice message for special characters")
	}

	// Verify subsequent valid inputs work
	if !strings.Contains(output, "moved successfully") {
		t.Fatalf("Expected move success after invalid input")
	}
	if !strings.Contains(output, "discarded (acked)") {
		t.Fatalf("Expected discard success after invalid input")
	}
}
