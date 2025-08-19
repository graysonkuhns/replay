package cmd_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"replay/constants"
	"replay/e2e_tests/testhelpers"

	"cloud.google.com/go/pubsub/v2"
)

func TestDLRWithPrettyJSON(t *testing.T) {
	t.Parallel()
	// Set up context and PubSub client.
	baseTest := testhelpers.NewBaseE2ETest(t, "dlr_pretty_json_test")

	// Create a JSON message for testing
	jsonData := map[string]interface{}{
		"id":        "test-123",
		"timestamp": time.Now().Format(time.RFC3339),
		"data": map[string]interface{}{
			"field1": "value1",
			"field2": 42,
			"nested": map[string]interface{}{
				"nestedField1": true,
				"nestedField2": []string{"item1", "item2"},
			},
		},
	}

	jsonBytes, err := json.Marshal(jsonData)
	if err != nil {
		t.Fatalf("Failed to marshal JSON data: %v", err)
	}

	// Create a pretty-printed version of the JSON to compare against
	prettyJSON, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal pretty JSON: %v", err)
	}

	// Publish the test message to the dead-letter topic
	message := pubsub.Message{
		Data: jsonBytes,
		Attributes: map[string]string{
			"testRun": baseTest.TestRunID,
		},
	}

	if err := baseTest.PublishAndWait([]pubsub.Message{message}); err != nil {
		t.Fatalf("Failed to publish test message: %v", err)
	}

	// Prepare CLI arguments for the dlr command with pretty-json flag.
	dlrArgs := []string{
		"dlr",
		"--source-type", constants.BrokerTypeGCPPubSubSubscription,
		"--destination-type", constants.BrokerTypeGCPPubSubTopic,
		"--source", baseTest.Setup.GetSourceSubscriptionName(),
		"--destination", baseTest.Setup.GetDestTopicName(),
		"--pretty-json", // Enable pretty JSON output
	}

	// Run the dlr command with custom args.
	actual, err := baseTest.RunDLRCommandWithArgs(dlrArgs, "m\n")
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}

	// Define expected output substrings.
	expectedLines := []string{
		fmt.Sprintf("Starting DLR review from %s", baseTest.Setup.GetSourceSubscriptionName()),
		"",
		"Message 1:",
		"Data (pretty JSON):",
		string(prettyJSON),
		fmt.Sprintf("Attributes: map[parallelIndex:%d testName:%s testRun:%s]", baseTest.TestContext.ParallelIndex, t.Name(), baseTest.TestRunID),
		"Choose action ([m]ove / [d]iscard / [q]uit): Message 1 moved successfully",
		"",
		"Dead-lettered messages review completed. Total messages processed: 1",
	}

	testhelpers.AssertCLIOutput(t, actual, expectedLines)

	t.Logf("Successfully verified pretty JSON output formatting")
}
