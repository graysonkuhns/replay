package cmd_test

import (
	"fmt"
	"io"
	"log"
	"os"
	"testing"
	"time"

	"replay/integration_tests/testhelpers"

	"cloud.google.com/go/pubsub"
)

func TestDLROperation(t *testing.T) {
	t.Parallel()
	// Set up integration test environment
	setup := testhelpers.SetupIntegrationTest(t)
	testRunValue := "dlr_test"

	// Purge the dead-letter (source) subscription.
	setup.PurgeSourceSubscription(t)

	// Publish two test messages to the dead-letter topic: one to move and one to discard.
	sourceTopic := setup.GetSourceTopic()
	messages := []pubsub.Message{
		{
			Data: []byte("Test message move"),
			Attributes: map[string]string{
				"testRun": testRunValue,
			},
		},
		{
			Data: []byte("Test message discard"),
			Attributes: map[string]string{
				"testRun": testRunValue,
			},
		},
	}
	_, err := testhelpers.PublishTestMessages(setup.Context, sourceTopic, messages, "test-ordering-key")
	if err != nil {
		t.Fatalf("Failed to publish test messages: %v", err)
	}

	// Wait for the messages to propagate to the dead-letter subscription.
	time.Sleep(10 * time.Second)

	// Prepare CLI arguments for the dlr command.
	dlrArgs := []string{
		"dlr",
		"--source-type", "GCP_PUBSUB_SUBSCRIPTION",
		"--destination-type", "GCP_PUBSUB_TOPIC",
		"--source", fmt.Sprintf("projects/%s/subscriptions/%s", setup.ProjectID, setup.SourceSubName),
		"--destination", fmt.Sprintf("projects/%s/topics/%s", setup.ProjectID, setup.DestTopicName),
	}

	// Simulate user inputs: "m" for moving message 1 and "d" for discarding message 2.
	origStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe for stdin: %v", err)
	}
	// Write simulated inputs and close the writer.
	_, err = io.WriteString(w, "m\nd\n")
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
		"Test message move",
		"Attributes: map[testRun:dlr_test]",
		"Choose action ([m]ove / [d]iscard / [q]uit): Message 1 moved successfully",
		"",
		"Message 2:",
		"Data:",
		"Test message discard",
		"Attributes: map[testRun:dlr_test]",
		"Choose action ([m]ove / [d]iscard / [q]uit): Message 2 discarded (acked)",
		"",
		"Dead-lettered messages review completed. Total messages processed: 2",
	}

	testhelpers.AssertCLIOutput(t, actual, expectedLines)

	// Allow time for the moved message to propagate.
	time.Sleep(5 * time.Second)

	// Verify the message was published to the destination topic via its subscription.
	received, err := testhelpers.PollMessages(setup.Context, setup.DestSub, testRunValue, 1)
	if err != nil {
		t.Fatalf("Error receiving messages: %v", err)
	}
	if len(received) != 1 {
		t.Fatalf("Expected 1 moved message, got %d", len(received))
	}
	log.Printf("Successfully moved and received %d message", len(received))

	// Verify that the discarded message is no longer in the source subscription.
	sourceReceived, err := testhelpers.PollMessages(setup.Context, setup.SourceSub, testRunValue, 0)
	if err != nil {
		t.Fatalf("Error polling source subscription: %v", err)
	}
	if len(sourceReceived) != 0 {
		t.Fatalf("Expected 0 messages in source subscription, got %d", len(sourceReceived))
	}
}
