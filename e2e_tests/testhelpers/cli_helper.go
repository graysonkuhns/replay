package testhelpers

import (
	"bytes"
	"os"
	"regexp"

	"replay/cmd"
)

// RunCLICommand sets up the CLI arguments, executes the CLI tool,
// captures both stdout and stderr output and replaces timestamps with "[TIMESTAMP]".
func RunCLICommand(args []string) (string, error) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = append([]string{"replay"}, args...)

	// Capture stdout using os.Pipe.
	rOut, wOut, err := os.Pipe()
	if err != nil {
		return "", err
	}
	oldOut := os.Stdout
	os.Stdout = wOut

	// Capture stderr using os.Pipe.
	rErr, wErr, err := os.Pipe()
	if err != nil {
		return "", err
	}
	oldErr := os.Stderr
	os.Stderr = wErr

	// Execute the CLI command.
	cmd.Execute()

	// Restore os.Stdout and os.Stderr.
	wOut.Close()
	wErr.Close()
	os.Stdout = oldOut
	os.Stderr = oldErr

	// Read from both pipes.
	var bufOut, bufErr bytes.Buffer
	if _, err := bufOut.ReadFrom(rOut); err != nil {
		rOut.Close()
		rErr.Close()
		return "", err
	}
	rOut.Close()

	if _, err := bufErr.ReadFrom(rErr); err != nil {
		rErr.Close()
		return "", err
	}
	rErr.Close()

	// Combine stdout and stderr output.
	output := bufOut.String() + bufErr.String()

	// Replace timestamp parts with a token.
	tsRe := regexp.MustCompile(`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}`)
	output = tsRe.ReplaceAllString(output, "[TIMESTAMP]")

	return output, nil
}
