package cmd_test

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"replay/integration_tests/testhelpers"

	"cloud.google.com/go/pubsub"
)

func TestDLRWithPrettyJSON(t *testing.T) {
	t.Parallel()
	// Set up context and PubSub client.
	setup := testhelpers.SetupIntegrationTest(t)
	testRunValue := "dlr_pretty_json_test"

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
	sourceTopic := setup.GetSourceTopic()
	message := pubsub.Message{
		Data: jsonBytes,
		Attributes: map[string]string{
			"testRun": testRunValue,
		},
	}

	_, err = testhelpers.PublishTestMessages(setup.Context, sourceTopic, []pubsub.Message{message}, "test-ordering-key")
	if err != nil {
		t.Fatalf("Failed to publish test message: %v", err)
	}

	// Wait for the message to propagate to the dead-letter subscription.
	time.Sleep(10 * time.Second)

	// Prepare CLI arguments for the dlr command with pretty-json flag.
	dlrArgs := []string{
		"dlr",
		"--source-type", "GCP_PUBSUB_SUBSCRIPTION",
		"--destination-type", "GCP_PUBSUB_TOPIC",
		"--source", fmt.Sprintf("projects/%s/subscriptions/%s", setup.ProjectID, setup.SourceSubName),
		"--destination", fmt.Sprintf("projects/%s/topics/%s", setup.ProjectID, setup.DestTopicName),
		"--pretty-json", // Enable pretty JSON output
	}

	// Simulate user input: "m" for moving the message
	origStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe for stdin: %v", err)
	}
	// Write simulated input and close the writer.
	_, err = io.WriteString(w, "m\n")
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
		"Data (pretty JSON):",
		string(prettyJSON),
		fmt.Sprintf("Attributes: map[testRun:%s]", testRunValue),
		"Choose action ([m]ove / [d]iscard / [q]uit): Message 1 moved successfully",
		"",
		"Dead-lettered messages review completed. Total messages processed: 1",
	}

	testhelpers.AssertCLIOutput(t, actual, expectedLines)

	t.Logf("Successfully verified pretty JSON output formatting")
}
