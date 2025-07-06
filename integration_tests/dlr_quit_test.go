package cmd_test

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"testing"
	"time"

	"replay/integration_tests/testhelpers"

	"cloud.google.com/go/pubsub"
)

func TestDLRQuitOperation(t *testing.T) {
	// Test to verify that the DLR operation correctly handles the quit command
	setup := testhelpers.SetupIntegrationTest(t)
	testRunValue := "dlr_quit_test"

	// Prepare 4 messages with unique body content.
	numMessages := 4
	sourceTopic := setup.GetSourceTopic()
	var messages []pubsub.Message
	orderingKey := "test-ordering-key"

	for i := 1; i <= numMessages; i++ {
		body := fmt.Sprintf("DLR Quit Test message %d", i)
		messages = append(messages, pubsub.Message{
			Data: []byte(body),
			Attributes: map[string]string{
				"testRun": testRunValue,
			},
		})
	}

	_, err := testhelpers.PublishTestMessages(setup.Context, sourceTopic, messages, orderingKey)
	if err != nil {
		t.Fatalf("Failed to publish test messages with ordering key: %v", err)
	}

	log.Printf("Published %d messages with ordering key: %s", numMessages, orderingKey)
	time.Sleep(15 * time.Second) // Wait for messages to arrive in the subscription

	// Prepare CLI arguments for the dlr command.
	dlrArgs := []string{
		"dlr",
		"--source-type", "GCP_PUBSUB_SUBSCRIPTION",
		"--destination-type", "GCP_PUBSUB_TOPIC",
		"--source", fmt.Sprintf("projects/%s/subscriptions/%s", setup.ProjectID, setup.SourceSubName),
		"--destination", fmt.Sprintf("projects/%s/topics/%s", setup.ProjectID, setup.DestTopicName),
	}

	// Simulate user inputs: "m" (move) for 2 messages, "d" (discard) for 1 message, then "q" (quit)
	origStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe for stdin: %v", err)
	}

	// Write inputs: move, move, discard, quit
	inputs := "m\nm\nd\nq\n"

	_, err = io.WriteString(w, inputs)
	if err != nil {
		t.Fatalf("Failed to write simulated input: %v", err)
	}
	w.Close()
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()

	// Run the dlr command.
	actual, err := testhelpers.RunCLICommand(dlrArgs)
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}

	// Define expected output substrings.
	expectedLines := []string{
		fmt.Sprintf("Starting DLR review from projects/%s/subscriptions/%s", setup.ProjectID, setup.SourceSubName),
		"",
		"Message 1:",
		"Data:",
		"DLR Quit Test message 1",
		"Attributes: map[testRun:dlr_quit_test]",
		"Choose action ([m]ove / [d]iscard / [q]uit): Message 1 moved successfully",
		"",
		"Message 2:",
		"Data:",
		"DLR Quit Test message 2",
		"Attributes: map[testRun:dlr_quit_test]",
		"Choose action ([m]ove / [d]iscard / [q]uit): Message 2 moved successfully",
		"",
		"Message 3:",
		"Data:",
		"DLR Quit Test message 3",
		"Attributes: map[testRun:dlr_quit_test]",
		"Choose action ([m]ove / [d]iscard / [q]uit): Message 3 discarded (acked)",
		"",
		"Message 4:",
		"Data:",
		"DLR Quit Test message 4",
		"Attributes: map[testRun:dlr_quit_test]",
		"Choose action ([m]ove / [d]iscard / [q]uit): Quitting review...",
		"",
		"Dead-lettered messages review completed. Total messages processed: 3",
	}

	testhelpers.AssertCLIOutput(t, actual, expectedLines)
	t.Logf("DLR command executed for quit operation test")

	// Allow time for moved messages to propagate.
	time.Sleep(5 * time.Second)

	// Poll the destination subscription for moved messages.
	// We expect exactly 2 messages to be moved.
	received, err := testhelpers.PollMessages(setup.Context, setup.DestSub, testRunValue, 2)
	if err != nil {
		t.Fatalf("Error receiving messages from destination: %v", err)
	}
	if len(received) != 2 {
		t.Fatalf("Expected 2 messages in destination, got %d", len(received))
	}

	// Verify correct bodies of moved messages
	expectedMovedMessages := []string{
		"DLR Quit Test message 1",
		"DLR Quit Test message 2",
	}

	for _, expected := range expectedMovedMessages {
		found := false
		for _, msg := range received {
			if string(msg.Data) == expected {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Expected moved message body '%s' not found in received messages", expected)
		}
	}

	// Wait for ack deadline to expire (10 seconds) before checking the source subscription
	time.Sleep(25 * time.Second)

	// Verify that one message remains in the source subscription (message 4)
	// We expect exactly 1 message to remain in the source subscription after processing.
	// Use a longer timeout context for this specific polling operation
	pollCtx, pollCancel := context.WithTimeout(setup.Context, 30*time.Second)
	defer pollCancel()
	sourceReceived, err := testhelpers.PollMessages(pollCtx, setup.SourceSub, testRunValue, 1)

	if err != nil {
		t.Fatalf("Error polling source subscription: %v", err)
	}
	if len(sourceReceived) != 1 {
		t.Fatalf("Expected 1 message in source subscription, got %d", len(sourceReceived))
	}

	// Verify the remaining message is the correct one (message 4 that we quit before processing)
	expectedRemainingMessage := "DLR Quit Test message 4"
	if string(sourceReceived[0].Data) != expectedRemainingMessage {
		t.Fatalf("Expected remaining message '%s', but got '%s'",
			expectedRemainingMessage, string(sourceReceived[0].Data))
	}

	t.Logf("Successfully verified DLR quit operation: 2 messages moved, 1 discarded, 1 remaining after quit")
}
