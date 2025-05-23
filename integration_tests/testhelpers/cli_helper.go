package testhelpers

import (
	"bytes"
	"os"
	"regexp"
	"replay/cmd"
)

// RunCLICommand sets up the CLI arguments, executes the CLI tool,
// captures its output and replaces timestamps with "[TIMESTAMP]".
func RunCLICommand(args []string) (string, error) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = append([]string{"replay"}, args...)

	// Capture CLI output using os.Pipe.
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	oldOut := os.Stdout
	os.Stdout = w

	// Execute the CLI command.
	cmd.Execute()

	// Restore os.Stdout.
	w.Close()
	os.Stdout = oldOut

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		r.Close()
		return "", err
	}
	r.Close()

	// Replace timestamp parts with a token.
	output := buf.String()
	tsRe := regexp.MustCompile(`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}`)
	output = tsRe.ReplaceAllString(output, "[TIMESTAMP]")

	return output, nil
}
