package cmd_test

import (
	"encoding/json"
	"testing"
	"time"

	"replay/integration_tests/testhelpers"
)

func TestDLRJSONMessageIntegrity(t *testing.T) {
	t.Parallel()
	setup := testhelpers.SetupIntegrationTest(t)
	testRunValue := "dlr_json_integrity_test"

	// Prepare JSON messages with various complexity levels using the builder.
	numMessages := 3
	sourceTopicName := setup.GetSourceTopicName()

	builder := testhelpers.NewTestMessageBuilder().
		WithAttributes(map[string]string{"testRun": testRunValue})

	var expectedJSONs []string

	// Use a fixed timestamp for deterministic testing
	fixedTimestamp := int64(1751547082)

	// Simple JSON
	simpleJSON := map[string]interface{}{
		"id":        1,
		"name":      "Simple JSON Message",
		"timestamp": fixedTimestamp,
		"isValid":   true,
	}
	builder.WithAttribute("contentType", "application/json").
		WithAttribute("messageIndex", "1").
		WithJSONMessage(simpleJSON)
	simpleJSONBytes, err := json.Marshal(simpleJSON)
	if err != nil {
		t.Fatalf("Failed to marshal simple JSON: %v", err)
	}
	expectedJSONs = append(expectedJSONs, string(simpleJSONBytes))

	// Nested JSON
	nestedJSON := map[string]interface{}{
		"id":        2,
		"name":      "Nested JSON Message",
		"timestamp": fixedTimestamp,
		"metadata": map[string]interface{}{
			"version": "1.0",
			"source":  "integration-test",
			"tags":    []string{"test", "json", "nested"},
		},
		"counts": []int{1, 2, 3, 4, 5},
	}
	builder.WithAttribute("messageIndex", "2").
		WithJSONMessage(nestedJSON)
	nestedJSONBytes, err := json.Marshal(nestedJSON)
	if err != nil {
		t.Fatalf("Failed to marshal nested JSON: %v", err)
	}
	expectedJSONs = append(expectedJSONs, string(nestedJSONBytes))

	// Complex JSON with special characters
	complexJSON := map[string]interface{}{
		"id":        3,
		"name":      "Complex JSON Message with special characters: !@#$%^&*()",
		"timestamp": fixedTimestamp,
		"data": map[string]interface{}{
			"items": []map[string]interface{}{
				{
					"id":          101,
					"value":       45.67,
					"enabled":     true,
					"description": "Item with unicode: 你好, こんにちは, 안녕하세요",
				},
				{
					"id":          102,
					"value":       -12.34,
					"enabled":     false,
					"description": "Item with emoji: 🚀 🔥 ⭐️ 🌈",
				},
			},
			"nullValue":   nil,
			"emptyArray":  []string{},
			"emptyObject": map[string]string{},
		},
	}
	builder.WithAttribute("messageIndex", "3").
		WithJSONMessage(complexJSON)
	complexJSONBytes, err := json.Marshal(complexJSON)
	if err != nil {
		t.Fatalf("Failed to marshal complex JSON: %v", err)
	}
	expectedJSONs = append(expectedJSONs, string(complexJSONBytes))

	messages := builder.Build()

	_, err = testhelpers.PublishTestMessages(setup.Context, setup.Client, sourceTopicName, messages, "json-test-ordering-key")
	if err != nil {
		t.Fatalf("Failed to publish JSON test messages: %v", err)
	}
	time.Sleep(30 * time.Second) // Wait for messages to arrive in the subscription

	// Prepare CLI arguments for the dlr command.
	dlrArgs := []string{
		"dlr",
		"--source-type", "GCP_PUBSUB_SUBSCRIPTION",
		"--destination-type", "GCP_PUBSUB_TOPIC",
		"--source", setup.GetSourceSubscriptionName(),
		"--destination", setup.GetDestTopicName(),
	}

	// Simulate user inputs: "m" (move) for all messages
	var inputs string
	for i := 0; i < numMessages; i++ {
		inputs += "m\n"
	}

	simulator, err := testhelpers.NewStdinSimulator(inputs)
	if err != nil {
		t.Fatalf("Failed to create stdin simulator: %v", err)
	}
	defer simulator.Cleanup()

	// Run the dlr command.
	_, err = testhelpers.RunCLICommand(dlrArgs)
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}

	t.Logf("DLR command executed for JSON integrity test")

	// Allow time for moved messages to propagate.
	time.Sleep(30 * time.Second)

	// Poll the destination subscription for moved messages.
	received, err := testhelpers.PollMessages(setup.Context, setup.Client, setup.GetDestSubscriptionName(), testRunValue, numMessages)
	if err != nil {
		t.Fatalf("Error receiving messages from destination: %v", err)
	}
	if len(received) != numMessages {
		t.Fatalf("Expected %d messages in destination, got %d", numMessages, len(received))
	}

	// Verify that each JSON message maintains its structure and values
	for _, expectedJSON := range expectedJSONs {
		found := false
		var expectedMap map[string]interface{}
		if err := json.Unmarshal([]byte(expectedJSON), &expectedMap); err != nil {
			t.Fatalf("Failed to unmarshal expected JSON: %v", err)
		}

		for _, msg := range received {
			var receivedMap map[string]interface{}
			if err := json.Unmarshal(msg.Data, &receivedMap); err != nil {
				// Skip messages that aren't valid JSON
				continue
			}

			// Compare ID field which should uniquely identify our test messages
			if receivedMap["id"] == expectedMap["id"] {
				found = true

				// Convert both to JSON strings for deep comparison
				expectedJSON, err := json.Marshal(expectedMap)
				if err != nil {
					t.Fatalf("Failed to marshal expected map: %v", err)
				}

				receivedJSON, err := json.Marshal(receivedMap)
				if err != nil {
					t.Fatalf("Failed to marshal received map: %v", err)
				}

				if string(expectedJSON) != string(receivedJSON) {
					t.Fatalf("JSON structure or values were altered during DLR operation.\nExpected: %s\nReceived: %s",
						string(expectedJSON), string(receivedJSON))
				}
				break
			}
		}

		if !found {
			t.Fatalf("Expected JSON message with ID %v not found in received messages", expectedMap["id"])
		}
	}

	// Verify that the source subscription is empty (all messages were moved)
	sourceReceived, err := testhelpers.PollMessages(setup.Context, setup.Client, setup.GetSourceSubscriptionName(), testRunValue, 0)
	if err != nil {
		t.Fatalf("Error polling source subscription: %v", err)
	}
	if len(sourceReceived) != 0 {
		t.Fatalf("Expected 0 messages in source subscription, got %d", len(sourceReceived))
	}

	t.Logf("JSON message integrity verified for all %d messages moved using DLR operation", numMessages)
}
