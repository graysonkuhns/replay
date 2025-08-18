package testhelpers

import (
	"io"
	"os"
	"sync"
)

// Global mutex to protect stdin manipulation across parallel tests
var stdinMutex sync.Mutex

// StdinSimulator provides a clean abstraction for simulating stdin input in tests.
// It handles all the complex pipe management and cleanup automatically.
//
// Example usage:
//
//	simulator, err := NewStdinSimulator("m\nd\nq\n")
//	if err != nil {
//	    t.Fatalf("Failed to create stdin simulator: %v", err)
//	}
//	defer simulator.Cleanup()
//
//	// Now run your CLI command that needs stdin input
//	output, err := RunCLICommand(args)
type StdinSimulator struct {
	original  *os.File
	reader    *os.File
	writer    *os.File
	mutexHeld bool // Track if this instance holds the mutex
}

// NewStdinSimulator creates a new stdin simulator with the provided input string.
// The inputs will be written to stdin when the CLI command reads from it.
// The caller must call Cleanup() to restore original stdin and prevent leaks.
// This function is thread-safe and uses a mutex to protect against concurrent stdin manipulation.
func NewStdinSimulator(inputs string) (*StdinSimulator, error) {
	// Acquire the mutex to ensure exclusive access to stdin
	stdinMutex.Lock()

	// Store the original stdin
	original := os.Stdin

	// Create a pipe for stdin simulation
	r, w, err := os.Pipe()
	if err != nil {
		stdinMutex.Unlock()
		return nil, err
	}

	// Write the simulated inputs to the pipe
	_, err = io.WriteString(w, inputs)
	if err != nil {
		// Clean up on error
		r.Close()
		w.Close()
		stdinMutex.Unlock()
		return nil, err
	}

	// Close the writer to signal EOF after inputs are written
	w.Close()

	// Replace stdin with the reader end of the pipe
	os.Stdin = r

	return &StdinSimulator{
		original:  original,
		reader:    r,
		writer:    nil, // Already closed
		mutexHeld: true,
	}, nil
}

// Cleanup restores the original stdin and closes the pipe resources.
// This should always be called, typically with defer, to prevent stdin leaks.
// This function releases the mutex that protects stdin manipulation.
func (s *StdinSimulator) Cleanup() {
	// Only perform cleanup if we hold the mutex
	if !s.mutexHeld {
		return
	}

	// Restore original stdin
	os.Stdin = s.original

	// Close the reader if it's still open
	if s.reader != nil {
		s.reader.Close()
		s.reader = nil
	}

	// Release the mutex to allow other tests to proceed
	stdinMutex.Unlock()
	s.mutexHeld = false
}
