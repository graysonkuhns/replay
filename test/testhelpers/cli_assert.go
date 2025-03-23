package testhelpers

import (
	"strings"
	"testing"
)

// AssertCLIOutput compares the actual output with the expected lines.
// If they don't match, it fails the test with a detailed message.
func AssertCLIOutput(t *testing.T, actual string, expectedLines []string) {
	expectedOutput := strings.Join(expectedLines, "\n") + "\n"
	if actual != expectedOutput {
		actualLines := strings.Split(strings.TrimSpace(actual), "\n")
		expectedStr := strings.Join(expectedLines, "\n")
		actualStr := strings.Join(actualLines, "\n")
		t.Fatalf("CLI output mismatch.\nExpected output:\n%s\nActual output:\n%s", expectedStr, actualStr)
	}
}
