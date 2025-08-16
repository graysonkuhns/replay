package testhelpers

import (
	"bufio"
	"os"
	"testing"
)

func TestStdinSimulator(t *testing.T) {
	// Test inputs to simulate
	testInputs := "first line\nsecond line\nthird line\n"

	// Create the simulator
	simulator, err := NewStdinSimulator(testInputs)
	if err != nil {
		t.Fatalf("Failed to create StdinSimulator: %v", err)
	}
	defer simulator.Cleanup()

	// Read from stdin to verify the simulator works
	scanner := bufio.NewScanner(os.Stdin)
	var receivedLines []string

	// Read all the simulated lines
	for scanner.Scan() {
		receivedLines = append(receivedLines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("Error reading from simulated stdin: %v", err)
	}

	// Verify we got the expected lines
	expectedLines := []string{"first line", "second line", "third line"}
	if len(receivedLines) != len(expectedLines) {
		t.Fatalf("Expected %d lines, got %d: %v", len(expectedLines), len(receivedLines), receivedLines)
	}

	for i, expected := range expectedLines {
		if receivedLines[i] != expected {
			t.Errorf("Line %d: expected %q, got %q", i+1, expected, receivedLines[i])
		}
	}
}

func TestStdinSimulatorCleanup(t *testing.T) {
	// Store original stdin
	originalStdin := os.Stdin

	// Create and use simulator
	simulator, err := NewStdinSimulator("test\n")
	if err != nil {
		t.Fatalf("Failed to create StdinSimulator: %v", err)
	}

	// Verify stdin was changed
	if os.Stdin == originalStdin {
		t.Error("Expected stdin to be changed after creating simulator")
	}

	// Cleanup
	simulator.Cleanup()

	// Verify stdin was restored
	if os.Stdin != originalStdin {
		t.Error("Expected stdin to be restored after cleanup")
	}
}

func TestStdinSimulatorErrorHandling(t *testing.T) {
	// Test with empty input (should work)
	simulator, err := NewStdinSimulator("")
	if err != nil {
		t.Fatalf("Failed to create StdinSimulator with empty input: %v", err)
	}
	simulator.Cleanup()

	// Test multiple cleanup calls (should not panic)
	simulator.Cleanup() // Should be safe to call multiple times
}
