package cmd_test

import (
	"encoding/json"
	"strings"
	"testing"

	"replay/integration_tests/testhelpers"
)

func TestDLRJSONMessageIntegrity(t *testing.T) {
	t.Parallel()
	baseTest := testhelpers.NewBaseIntegrationTest(t, "dlr_json_integrity_test")

	// Prepare JSON messages with various complexity levels using the builder.
	numMessages := 3

	builder := testhelpers.NewTestMessageBuilder().
		WithAttributes(map[string]string{"testRun": baseTest.TestRunID})

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

	if err := baseTest.PublishAndWait(messages); err != nil {
		t.Fatalf("Failed to publish JSON test messages: %v", err)
	}

	// Simulate user inputs: "m" (move) for all messages
	inputs := strings.Repeat("m\n", numMessages)

	// Run the dlr command.
	_, err = baseTest.RunDLRCommand(inputs)
	if err != nil {
		t.Fatalf("Error running CLI command: %v", err)
	}

	t.Logf("DLR command executed for JSON integrity test")

	// Allow time for moved messages to propagate.
	baseTest.WaitForMessagePropagation()

	// Poll the destination subscription for moved messages.
	received, err := baseTest.GetMessagesFromDestination(numMessages)
	if err != nil {
		t.Fatalf("Error receiving messages from destination: %v", err)
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
	if err := baseTest.VerifyMessagesInSource(0); err != nil {
		t.Fatalf("%v", err)
	}

	t.Logf("JSON message integrity verified for all %d messages moved using DLR operation", numMessages)
}
