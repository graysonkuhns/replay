package cmd_test

import (
	"fmt"
	"log"
	"testing"
	"time"

	"cloud.google.com/go/pubsub"

	"replay/integration_tests/testhelpers"
)

func init() {
	log.Printf("Test suite initialization: logs are enabled")
}

func TestMoveStopsWhenSourceExhausted(t *testing.T) {
	log.Printf("Starting TestMoveStopsWhenSourceExhausted: verifying stop when source runs out of messages")
	setup := testhelpers.SetupIntegrationTest(t)
	// For this test, we move messages from the dead letter infrastructure to the normal events infrastructure.

	setup.PurgeSubscriptions(t)
	log.Printf("Completed purge of subscriptions")

	sourceTopic := setup.GetSourceTopic()
	numMessages := 3
	testRunValue := "move_test"

	var messages []pubsub.Message
	for i := 1; i <= numMessages; i++ {
		messages = append(messages, pubsub.Message{
			Data: []byte(fmt.Sprintf("Test message %d", i)),
			Attributes: map[string]string{
				"testRun": testRunValue,
			},
		})
	}

	_, err := testhelpers.PublishTestMessages(setup.Context, sourceTopic, messages, "test-ordering-key")
	if err != nil {
		t.Fatalf("Failed to publish test messages: %v", err)
	}

	time.Sleep(10 * time.Second)
	log.Printf("Completed waiting for dead letter subscription to receive messages")

	moveArgs := []string{
		"move",
		"--source-type", "GCP_PUBSUB_SUBSCRIPTION",
		"--destination-type", "GCP_PUBSUB_TOPIC",
		"--source", fmt.Sprintf("projects/%s/subscriptions/%s", setup.ProjectID, setup.SourceSubName),
		"--destination", fmt.Sprintf("projects/%s/topics/%s", setup.ProjectID, setup.DestTopicName),
	}

	actual, err := testhelpers.RunCLICommand(moveArgs)
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}
	log.Printf("Move command executed")

	// Define expected output lines.
	expectedLines := []string{
		fmt.Sprintf("[TIMESTAMP] Moving messages from projects/%s/subscriptions/%s to projects/%s/topics/%s", setup.ProjectID, setup.SourceSubName, setup.ProjectID, setup.DestTopicName),
		"[TIMESTAMP] Pulled message 1",
		"[TIMESTAMP] Publishing message 1",
		"[TIMESTAMP] Published message 1 successfully",
		"[TIMESTAMP] Acked message 1",
		"[TIMESTAMP] Processed message 1",
		"[TIMESTAMP] Pulled message 2",
		"[TIMESTAMP] Publishing message 2",
		"[TIMESTAMP] Published message 2 successfully",
		"[TIMESTAMP] Acked message 2",
		"[TIMESTAMP] Processed message 2",
		"[TIMESTAMP] Pulled message 3",
		"[TIMESTAMP] Publishing message 3",
		"[TIMESTAMP] Published message 3 successfully",
		"[TIMESTAMP] Acked message 3",
		"[TIMESTAMP] Processed message 3",
		"[TIMESTAMP] No messages received within timeout",
		fmt.Sprintf("[TIMESTAMP] Move operation completed. Total messages moved: %d", numMessages),
	}

	testhelpers.AssertCLIOutput(t, actual, expectedLines)

	time.Sleep(5 * time.Second)
	log.Printf("Waiting for messages to propagate to destination subscription")

	log.Printf("Starting to receive messages from destination subscription: default-events-subscription")
	received, err := testhelpers.PollMessages(setup.Context, setup.DestSub, testRunValue, numMessages)
	if err != nil {
		t.Fatalf("Error receiving messages: %v", err)
	}
	log.Printf("Successfully received %d messages", len(received))

	if len(received) != numMessages {
		t.Fatalf("Expected %d moved messages, got %d", numMessages, len(received))
	}

	log.Printf("Successfully received %d messages", len(received))
	t.Logf("Successfully moved %d messages", numMessages)
}

func TestMoveOperationWithCount(t *testing.T) {
	log.Printf("Starting TestMoveOperationWithCount")
	setup := testhelpers.SetupIntegrationTest(t)

	// Create client and purge subscriptions.
	setup.PurgeSubscriptions(t)

	numMessages := 5
	moveCount := 3
	testRunValue := "move_test_count"
	sourceTopic := setup.GetSourceTopic()

	var messages []pubsub.Message
	for i := 1; i <= numMessages; i++ {
		messages = append(messages, pubsub.Message{
			Data: []byte(fmt.Sprintf("Count Test message %d", i)),
			Attributes: map[string]string{
				"testRun": testRunValue,
			},
		})
	}

	_, err := testhelpers.PublishTestMessages(setup.Context, sourceTopic, messages, "test-ordering-key")
	if err != nil {
		t.Fatalf("Failed to publish test messages: %v", err)
	}

	time.Sleep(10 * time.Second)
	log.Printf("Completed waiting for messages to be available in the source subscription")

	moveArgs := []string{
		"move",
		"--source-type", "GCP_PUBSUB_SUBSCRIPTION",
		"--destination-type", "GCP_PUBSUB_TOPIC",
		"--source", fmt.Sprintf("projects/%s/subscriptions/%s", setup.ProjectID, setup.SourceSubName),
		"--destination", fmt.Sprintf("projects/%s/topics/%s", setup.ProjectID, setup.DestTopicName),
		"--count", fmt.Sprintf("%d", moveCount),
	}

	actual, err := testhelpers.RunCLICommand(moveArgs)
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}
	log.Printf("Move command executed with count %d", moveCount)

	expectedLines := []string{
		fmt.Sprintf("[TIMESTAMP] Moving messages from projects/%s/subscriptions/%s to projects/%s/topics/%s", setup.ProjectID, setup.SourceSubName, setup.ProjectID, setup.DestTopicName),
		"[TIMESTAMP] Pulled message 1",
		"[TIMESTAMP] Publishing message 1",
		"[TIMESTAMP] Published message 1 successfully",
		"[TIMESTAMP] Acked message 1",
		"[TIMESTAMP] Processed message 1",
		"[TIMESTAMP] Pulled message 2",
		"[TIMESTAMP] Publishing message 2",
		"[TIMESTAMP] Published message 2 successfully",
		"[TIMESTAMP] Acked message 2",
		"[TIMESTAMP] Processed message 2",
		"[TIMESTAMP] Pulled message 3",
		"[TIMESTAMP] Publishing message 3",
		"[TIMESTAMP] Published message 3 successfully",
		"[TIMESTAMP] Acked message 3",
		"[TIMESTAMP] Processed message 3",
		fmt.Sprintf("[TIMESTAMP] Move operation completed. Total messages moved: %d", moveCount),
	}

	testhelpers.AssertCLIOutput(t, actual, expectedLines)

	time.Sleep(5 * time.Second)
	log.Printf("Polling destination subscription for moved messages")
	movedMessages, err := testhelpers.PollMessages(setup.Context, setup.DestSub, testRunValue, moveCount)
	if err != nil {
		t.Fatalf("Error receiving moved messages: %v", err)
	}
	if len(movedMessages) != moveCount {
		t.Fatalf("Expected %d moved messages in destination, got %d", moveCount, len(movedMessages))
	}

	log.Printf("Polling source subscription for remaining messages")
	remainingMessages, err := testhelpers.PollMessages(setup.Context, setup.SourceSub, testRunValue, numMessages-moveCount)
	if err != nil {
		t.Fatalf("Error receiving remaining messages: %v", err)
	}
	if len(remainingMessages) != numMessages-moveCount {
		t.Fatalf("Expected %d remaining messages in source, got %d", numMessages-moveCount, len(remainingMessages))
	}

	t.Logf("Successfully moved %d messages and found %d remaining in source", moveCount, numMessages-moveCount)
}

func TestMoveMessageBodyIntegrity(t *testing.T) {
	// New test to verify that the body content of moved messages remains unchanged.
	setup := testhelpers.SetupIntegrationTest(t)
	// Use same names as other tests.

	// Purge subscriptions.
	setup.PurgeSubscriptions(t)

	// Prepare messages with unique body content.
	numMessages := 3
	testRunValue := "move_test_body_integrity"
	sourceTopic := setup.GetSourceTopic()
	var messages []pubsub.Message
	var expectedBodies []string
	for i := 1; i <= numMessages; i++ {
		body := fmt.Sprintf("Integrity Test message %d", i)
		expectedBodies = append(expectedBodies, body)
		messages = append(messages, pubsub.Message{
			Data: []byte(body),
			Attributes: map[string]string{
				"testRun": testRunValue,
			},
		})
	}

	_, err := testhelpers.PublishTestMessages(setup.Context, sourceTopic, messages, "test-ordering-key")
	if err != nil {
		t.Fatalf("Failed to publish test messages: %v", err)
	}
	// Increase sleep duration to 15 seconds to ensure all messages arrive.
	time.Sleep(15 * time.Second)

	// Run the move command.
	moveArgs := []string{
		"move",
		"--source-type", "GCP_PUBSUB_SUBSCRIPTION",
		"--destination-type", "GCP_PUBSUB_TOPIC",
		"--source", fmt.Sprintf("projects/%s/subscriptions/%s", setup.ProjectID, setup.SourceSubName),
		"--destination", fmt.Sprintf("projects/%s/topics/%s", setup.ProjectID, setup.DestTopicName),
		"--count", fmt.Sprintf("%d", numMessages),
	}
	actual, err := testhelpers.RunCLICommand(moveArgs)
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}
	t.Logf("Move command executed for body integrity test: %s", actual)

	// Poll the destination subscription.
	received, err := testhelpers.PollMessages(setup.Context, setup.DestSub, testRunValue, numMessages)
	if err != nil {
		t.Fatalf("Error receiving messages from destination: %v", err)
	}
	if len(received) != numMessages {
		t.Fatalf("Expected %d messages in destination, got %d", numMessages, len(received))
	}

	// Verify that each expected message body is found in the received messages, regardless of order.
	for _, expected := range expectedBodies {
		found := false
		for _, msg := range received {
			if string(msg.Data) == expected {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Expected message body '%s' not found in received messages", expected)
		}
	}
	t.Logf("Message body integrity verified for all %d messages", numMessages)
}
