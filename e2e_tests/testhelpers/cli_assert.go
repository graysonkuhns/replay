package testhelpers

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"cloud.google.com/go/pubsub/v2"
)

// AssertCLIOutput compares the actual output with the expected lines.
// If they don't match, it fails the test with a detailed message.
func AssertCLIOutput(t testing.TB, actual string, expectedLines []string) {
	expectedOutput := strings.Join(expectedLines, "\n") + "\n"
	if actual != expectedOutput {
		actualLines := strings.Split(strings.TrimSpace(actual), "\n")
		expectedStr := strings.Join(expectedLines, "\n")
		actualStr := strings.Join(actualLines, "\n")
		t.Fatalf(
			"CLI output mismatch.\n"+
				"===== Start EXPECTED output =====\n%s\n===== End EXPECTED output =====\n"+
				"===== Start ACTUAL output =====\n%s\n===== End ACTUAL output =====\n",
			expectedStr,
			actualStr,
		)
	}
}

// AssertMessageContent compares the actual message content with the expected content.
// If they don't match, it fails the test with a detailed message.
func AssertMessageContent(t testing.TB, actual, expected string) {
	t.Helper()
	if actual != expected {
		t.Fatalf(
			"Message content mismatch.\n"+
				"Expected: %q\n"+
				"Actual:   %q\n",
			expected,
			actual,
		)
	}
}

// AssertMessageCount compares the actual number of messages with the expected count.
// If they don't match, it fails the test with a detailed message.
func AssertMessageCount(t testing.TB, messages []*pubsub.Message, expected int) {
	t.Helper()
	actual := len(messages)
	if actual != expected {
		t.Fatalf(
			"Message count mismatch.\n"+
				"Expected: %d messages\n"+
				"Actual:   %d messages\n",
			expected,
			actual,
		)
	}
}

// AssertJSONEquals compares two JSON byte arrays for equality.
// It unmarshals both and compares them as objects to ignore formatting differences.
// If they don't match, it fails the test with a detailed message.
func AssertJSONEquals(t testing.TB, actual, expected []byte) {
	t.Helper()

	var actualObj, expectedObj interface{}

	if err := json.Unmarshal(actual, &actualObj); err != nil {
		t.Fatalf("Failed to unmarshal actual JSON: %v\nJSON: %s", err, string(actual))
	}

	if err := json.Unmarshal(expected, &expectedObj); err != nil {
		t.Fatalf("Failed to unmarshal expected JSON: %v\nJSON: %s", err, string(expected))
	}

	if !reflect.DeepEqual(actualObj, expectedObj) {
		// Pretty print both for better error messages
		actualPretty, _ := json.MarshalIndent(actualObj, "", "  ")
		expectedPretty, _ := json.MarshalIndent(expectedObj, "", "  ")

		t.Fatalf(
			"JSON content mismatch.\n"+
				"===== Start EXPECTED JSON =====\n%s\n===== End EXPECTED JSON =====\n"+
				"===== Start ACTUAL JSON =====\n%s\n===== End ACTUAL JSON =====\n",
			string(expectedPretty),
			string(actualPretty),
		)
	}
}

// AssertBinaryEquals compares two binary byte arrays for equality.
// If they don't match, it fails the test with a detailed message.
func AssertBinaryEquals(t testing.TB, actual, expected []byte) {
	t.Helper()
	if !bytes.Equal(actual, expected) {
		t.Fatalf(
			"Binary content mismatch.\n"+
				"Expected length: %d bytes\n"+
				"Actual length:   %d bytes\n"+
				"Expected (hex):  %x\n"+
				"Actual (hex):    %x\n",
			len(expected),
			len(actual),
			expected,
			actual,
		)
	}
}

// AssertContainsInOrder checks if the output contains all expected strings in the given order.
// The strings don't need to be consecutive, but they must appear in the specified order.
// If they don't appear in order, it fails the test with a detailed message.
func AssertContainsInOrder(t testing.TB, output string, expectedStrings []string) {
	t.Helper()

	if len(expectedStrings) == 0 {
		return // Nothing to check
	}

	lines := strings.Split(output, "\n")
	lineIndex := 0
	expectedIndex := 0

	for lineIndex < len(lines) && expectedIndex < len(expectedStrings) {
		if strings.Contains(lines[lineIndex], expectedStrings[expectedIndex]) {
			expectedIndex++
		}
		lineIndex++
	}

	if expectedIndex < len(expectedStrings) {
		// Find which strings were not found
		notFound := expectedStrings[expectedIndex:]

		t.Fatalf(
			"Output does not contain all expected strings in order.\n"+
				"Missing strings starting from: %q\n"+
				"Not found: %v\n"+
				"===== Start OUTPUT =====\n%s\n===== End OUTPUT =====\n",
			expectedStrings[expectedIndex],
			notFound,
			output,
		)
	}
}
