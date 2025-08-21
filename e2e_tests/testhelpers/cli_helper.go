package testhelpers

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
)

// RunCLICommand executes the replay CLI binary as a subprocess,
// captures both stdout and stderr output and replaces timestamps with "[TIMESTAMP]".
func RunCLICommand(args []string) (string, error) {
	// Look for the binary in the workspace root
	workspaceRoot := os.Getenv("REPLAY_WORKSPACE_ROOT")
	if workspaceRoot == "" {
		// If not set, try to find it relative to the test file
		workspaceRoot = filepath.Join("..", "..")
	}

	binaryPath := filepath.Join(workspaceRoot, "replay")

	// Check if binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return "", fmt.Errorf("replay binary not found at %s. Please run 'go build' first", binaryPath)
	}

	// Create the command
	cmd := exec.Command(binaryPath, args...)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set stdin to os.Stdin to allow interactive commands
	cmd.Stdin = os.Stdin

	// Execute the command
	err := cmd.Run()

	// Combine stdout and stderr output
	output := stdout.String() + stderr.String()

	// Replace timestamp parts with a token
	tsRe := regexp.MustCompile(`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}`)
	output = tsRe.ReplaceAllString(output, "[TIMESTAMP]")

	// Return output even if command failed (for testing error cases)
	return output, err
}
