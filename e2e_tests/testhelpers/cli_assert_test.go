package testhelpers

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"cloud.google.com/go/pubsub/v2"
)

// MockTestingT is a mock implementation of testing.TB for testing our assertions
type MockTestingT struct {
	testing.TB // Embed testing.TB interface
	failed     bool
	message    string
}

func (m *MockTestingT) Fatalf(format string, args ...interface{}) {
	m.failed = true
	// Format the message with the provided arguments
	formattedMessage := format
	if len(args) > 0 {
		formattedMessage = fmt.Sprintf(format, args...)
	}
	m.message = strings.TrimSpace(strings.ReplaceAll(formattedMessage, "\n", " "))
}

func (m *MockTestingT) Helper() {}

// Implement remaining testing.TB methods as no-ops
func (m *MockTestingT) Error(args ...interface{})                 {}
func (m *MockTestingT) Errorf(format string, args ...interface{}) {}
func (m *MockTestingT) Fail()                                     {}
func (m *MockTestingT) FailNow()                                  {}
func (m *MockTestingT) Failed() bool                              { return m.failed }
func (m *MockTestingT) Fatal(args ...interface{})                 { m.failed = true }
func (m *MockTestingT) Log(args ...interface{})                   {}
func (m *MockTestingT) Logf(format string, args ...interface{})   {}
func (m *MockTestingT) Name() string                              { return "MockTest" }
func (m *MockTestingT) Skip(args ...interface{})                  {}
func (m *MockTestingT) SkipNow()                                  {}
func (m *MockTestingT) Skipf(format string, args ...interface{})  {}
func (m *MockTestingT) Skipped() bool                             { return false }
func (m *MockTestingT) TempDir() string                           { return "" }
func (m *MockTestingT) Cleanup(func())                            {}
func (m *MockTestingT) Setenv(key, value string)                  {}

func TestAssertCLIOutput(t *testing.T) {
	tests := []struct {
		name          string
		actual        string
		expectedLines []string
		shouldFail    bool
	}{
		{
			name:          "matching output",
			actual:        "line1\nline2\nline3\n",
			expectedLines: []string{"line1", "line2", "line3"},
			shouldFail:    false,
		},
		{
			name:          "mismatching output",
			actual:        "line1\nline2\n",
			expectedLines: []string{"line1", "line2", "line3"},
			shouldFail:    true,
		},
		{
			name:          "empty expected",
			actual:        "\n",
			expectedLines: []string{},
			shouldFail:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockTestingT{}
			AssertCLIOutput(mock, tt.actual, tt.expectedLines)

			if mock.failed != tt.shouldFail {
				t.Errorf("AssertCLIOutput failed=%v, want %v", mock.failed, tt.shouldFail)
			}
		})
	}
}

func TestAssertMessageContent(t *testing.T) {
	tests := []struct {
		name       string
		actual     string
		expected   string
		shouldFail bool
	}{
		{
			name:       "matching content",
			actual:     "Hello, World!",
			expected:   "Hello, World!",
			shouldFail: false,
		},
		{
			name:       "mismatching content",
			actual:     "Hello, World!",
			expected:   "Hello, Universe!",
			shouldFail: true,
		},
		{
			name:       "empty strings match",
			actual:     "",
			expected:   "",
			shouldFail: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockTestingT{}
			AssertMessageContent(mock, tt.actual, tt.expected)

			if mock.failed != tt.shouldFail {
				t.Errorf("AssertMessageContent failed=%v, want %v", mock.failed, tt.shouldFail)
			}
		})
	}
}

func TestAssertMessageCount(t *testing.T) {
	tests := []struct {
		name       string
		messages   []*pubsub.Message
		expected   int
		shouldFail bool
	}{
		{
			name: "matching count",
			messages: []*pubsub.Message{
				{Data: []byte("msg1")},
				{Data: []byte("msg2")},
			},
			expected:   2,
			shouldFail: false,
		},
		{
			name: "mismatching count",
			messages: []*pubsub.Message{
				{Data: []byte("msg1")},
			},
			expected:   2,
			shouldFail: true,
		},
		{
			name:       "empty messages",
			messages:   []*pubsub.Message{},
			expected:   0,
			shouldFail: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockTestingT{}
			AssertMessageCount(mock, tt.messages, tt.expected)

			if mock.failed != tt.shouldFail {
				t.Errorf("AssertMessageCount failed=%v, want %v", mock.failed, tt.shouldFail)
			}
		})
	}
}

func TestAssertJSONEquals(t *testing.T) {
	tests := []struct {
		name       string
		actual     []byte
		expected   []byte
		shouldFail bool
	}{
		{
			name:       "matching JSON objects",
			actual:     []byte(`{"name":"John","age":30}`),
			expected:   []byte(`{"age":30,"name":"John"}`), // Different order but same content
			shouldFail: false,
		},
		{
			name:       "matching JSON with different formatting",
			actual:     []byte(`{"name": "John", "age": 30}`),
			expected:   []byte(`{"name":"John","age":30}`),
			shouldFail: false,
		},
		{
			name:       "mismatching JSON",
			actual:     []byte(`{"name":"John","age":30}`),
			expected:   []byte(`{"name":"Jane","age":25}`),
			shouldFail: true,
		},
		{
			name:       "invalid actual JSON",
			actual:     []byte(`{invalid json}`),
			expected:   []byte(`{"valid":"json"}`),
			shouldFail: true,
		},
		{
			name:       "invalid expected JSON",
			actual:     []byte(`{"valid":"json"}`),
			expected:   []byte(`{invalid json}`),
			shouldFail: true,
		},
		{
			name:       "matching arrays",
			actual:     []byte(`[1,2,3]`),
			expected:   []byte(`[1, 2, 3]`),
			shouldFail: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockTestingT{}
			AssertJSONEquals(mock, tt.actual, tt.expected)

			if mock.failed != tt.shouldFail {
				t.Errorf("AssertJSONEquals failed=%v, want %v", mock.failed, tt.shouldFail)
			}
		})
	}
}

func TestAssertBinaryEquals(t *testing.T) {
	tests := []struct {
		name       string
		actual     []byte
		expected   []byte
		shouldFail bool
	}{
		{
			name:       "matching binary data",
			actual:     []byte{0x01, 0x02, 0x03, 0x04},
			expected:   []byte{0x01, 0x02, 0x03, 0x04},
			shouldFail: false,
		},
		{
			name:       "mismatching binary data",
			actual:     []byte{0x01, 0x02, 0x03},
			expected:   []byte{0x01, 0x02, 0x04},
			shouldFail: true,
		},
		{
			name:       "different lengths",
			actual:     []byte{0x01, 0x02},
			expected:   []byte{0x01, 0x02, 0x03},
			shouldFail: true,
		},
		{
			name:       "empty binary data",
			actual:     []byte{},
			expected:   []byte{},
			shouldFail: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockTestingT{}
			AssertBinaryEquals(mock, tt.actual, tt.expected)

			if mock.failed != tt.shouldFail {
				t.Errorf("AssertBinaryEquals failed=%v, want %v", mock.failed, tt.shouldFail)
			}
		})
	}
}

func TestAssertContainsInOrder(t *testing.T) {
	tests := []struct {
		name            string
		output          string
		expectedStrings []string
		shouldFail      bool
	}{
		{
			name:            "all strings in order",
			output:          "line1\nline2\nline3\nline4\n",
			expectedStrings: []string{"line1", "line3", "line4"},
			shouldFail:      false,
		},
		{
			name:            "strings out of order",
			output:          "line1\nline2\nline3\nline4\n",
			expectedStrings: []string{"line3", "line1"},
			shouldFail:      true,
		},
		{
			name:            "missing string",
			output:          "line1\nline2\nline3\n",
			expectedStrings: []string{"line1", "line4"},
			shouldFail:      true,
		},
		{
			name:            "empty expected strings",
			output:          "line1\nline2\n",
			expectedStrings: []string{},
			shouldFail:      false,
		},
		{
			name:            "substring matches",
			output:          "This is line one\nThis is line two\nThis is line three\n",
			expectedStrings: []string{"line one", "line three"},
			shouldFail:      false,
		},
		{
			name:            "case sensitive matching",
			output:          "Line1\nLine2\n",
			expectedStrings: []string{"line1"},
			shouldFail:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockTestingT{}
			AssertContainsInOrder(mock, tt.output, tt.expectedStrings)

			if mock.failed != tt.shouldFail {
				t.Errorf("AssertContainsInOrder failed=%v, want %v", mock.failed, tt.shouldFail)
			}
		})
	}
}

// Test the actual error messages for better coverage
func TestAssertMessageContentErrorMessage(t *testing.T) {
	mock := &MockTestingT{}
	AssertMessageContent(mock, "actual", "expected")

	if !strings.Contains(mock.message, "Message content mismatch") {
		t.Errorf("Error message should contain 'Message content mismatch', got: %s", mock.message)
	}
}

func TestAssertJSONEqualsWithComplexStructures(t *testing.T) {
	// Test with nested objects
	actual := []byte(`{
		"user": {
			"name": "John",
			"addresses": [
				{"street": "123 Main St", "city": "NYC"},
				{"street": "456 Elm St", "city": "LA"}
			]
		}
	}`)

	expected := []byte(`{"user":{"name":"John","addresses":[{"street":"123 Main St","city":"NYC"},{"street":"456 Elm St","city":"LA"}]}}`)

	mock := &MockTestingT{}
	AssertJSONEquals(mock, actual, expected)

	if mock.failed {
		t.Errorf("AssertJSONEquals should not fail for equivalent nested JSON structures")
	}
}

// Test for edge cases
func TestAssertBinaryEqualsWithNil(t *testing.T) {
	// Test with nil vs empty slice
	mock := &MockTestingT{}
	AssertBinaryEquals(mock, nil, []byte{})

	if mock.failed {
		t.Errorf("AssertBinaryEquals should treat nil and empty slice as equal")
	}
}

// Additional test for bytes.Equal compatibility
func TestAssertBinaryEqualsCompatibility(t *testing.T) {
	actual := []byte("Hello, World!")
	expected := []byte("Hello, World!")

	// Verify our function behaves like bytes.Equal
	if !bytes.Equal(actual, expected) {
		t.Fatal("Test setup error: bytes should be equal")
	}

	mock := &MockTestingT{}
	AssertBinaryEquals(mock, actual, expected)

	if mock.failed {
		t.Errorf("AssertBinaryEquals should not fail when bytes.Equal returns true")
	}
}

// Test JSON unmarshaling error handling
// For simplicity, we'll just test that invalid JSON triggers a failure
// rather than checking the exact error message, since MockTestingT doesn't
// stop execution like real testing.T
func TestAssertJSONEqualsUnmarshalError(t *testing.T) {
	// Since json.Unmarshal can be very forgiving, let's just ensure
	// that our assertion function fails when given clearly invalid input

	// The actual behavior is that Fatalf would stop execution immediately,
	// but our mock doesn't, so we'll just verify it was called
	actualMock := &MockTestingT{}
	AssertJSONEquals(actualMock, []byte(`{]`), []byte(`{"valid": "json"}`))

	if !actualMock.failed {
		t.Error("AssertJSONEquals should fail when actual has invalid JSON syntax")
	}

	// For the expected JSON error
	expectedMock := &MockTestingT{}
	AssertJSONEquals(expectedMock, []byte(`{"valid": "json"}`), []byte(`{]`))

	if !expectedMock.failed {
		t.Error("AssertJSONEquals should fail when expected has invalid JSON syntax")
	}
}

// Test JSON with special types
func TestAssertJSONEqualsWithNumbers(t *testing.T) {
	// JSON numbers are parsed as float64 by default
	actual := []byte(`{"count": 10, "price": 99.99}`)
	expected := []byte(`{"count": 10.0, "price": 99.99}`)

	mock := &MockTestingT{}
	AssertJSONEquals(mock, actual, expected)

	if mock.failed {
		t.Errorf("AssertJSONEquals should handle number formatting differences")
	}
}
