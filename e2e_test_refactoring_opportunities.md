# E2E Test Refactoring Opportunities

This document outlines potential improvements to the e2e test suite for the replay CLI tool. The analysis is based on examining the current test structure, patterns, and implementation details.

## 1. Replace Hard-coded Sleeps with Smart Polling

### Current Issue
Tests use multiple `time.Sleep()` calls with arbitrary durations (20s, 30s, 70s) to wait for messages to propagate through Pub/Sub.

```go
time.Sleep(20 * time.Second) // Wait for messages to propagate
time.Sleep(30 * time.Second) // Wait for messages to arrive
time.Sleep(70 * time.Second) // Wait for ack deadline to expire
```

### Recommended Approach
- Implement a `WaitForCondition` helper that polls with exponential backoff
- Create specific waiting functions like `WaitForMessagesInSubscription`
- Use context with timeout for better control
- Consider using Pub/Sub's acknowledgment deadline API to check message states

### Benefits
- Faster tests (no unnecessary waiting)
- More reliable (less timing-dependent)
- Better debugging (can log what we're waiting for)

## 2. Create a Test Suite Base Structure

### Current Issue
Each test file has similar setup patterns but implements them independently.

### Recommended Approach
Create a `BaseE2ETest` struct that encapsulates common functionality:

```go
type BaseE2ETest struct {
    *testing.T
    Setup *testhelpers.TestSetup
    TestRunID string
}

func (b *BaseE2ETest) PublishMessages(messages []pubsub.Message) error
func (b *BaseE2ETest) RunDLRCommand(inputs string) (string, error)
func (b *BaseE2ETest) RunMoveCommand(count int) (string, error)
func (b *BaseE2ETest) VerifyMessagesInDestination(expected int) error
func (b *BaseE2ETest) VerifyMessagesInSource(expected int) error
```

### Benefits
- Reduced code duplication
- Consistent test patterns
- Easier to add new tests
- Single place to update common behavior

## 3. Abstract stdin Simulation [COMPLETED]

### Current Issue
Every test that simulates user input has complex pipe management code:

```go
origStdin := os.Stdin
r, w, err := os.Pipe()
// ... setup and cleanup
```

### Recommended Approach
Create a `StdinSimulator` helper:

```go
type StdinSimulator struct {
    original *os.File
    reader   *os.File
    writer   *os.File
}

func NewStdinSimulator(inputs string) (*StdinSimulator, error)
func (s *StdinSimulator) Cleanup()
```

### Benefits
- Cleaner test code
- Consistent stdin handling
- Easier to test interactive commands
- Reduced chance of stdin leaks

## 4. Improve Test Data Management [COMPLETED]

### Current Issue
Test data is created inline with hard-coded values scattered throughout tests.

### Recommended Approach
Create test data builders:

```go
type TestMessageBuilder struct {
    messages []pubsub.Message
}

func (b *TestMessageBuilder) WithTextMessage(content string) *TestMessageBuilder
func (b *TestMessageBuilder) WithJSONMessage(data interface{}) *TestMessageBuilder
func (b *TestMessageBuilder) WithBinaryMessage(size int) *TestMessageBuilder
func (b *TestMessageBuilder) WithAttributes(attrs map[string]string) *TestMessageBuilder
func (b *TestMessageBuilder) Build() []pubsub.Message
```

### Benefits
- Reusable test data patterns
- Easier to create complex test scenarios
- Clear intent in tests
- Centralized test data generation

## 5. Better Assertion Helpers

### Current Issue
Mix of custom assertions and standard `t.Fatal` calls with inconsistent error messages.

### Recommended Approach
Extend the assertion helpers to cover more cases:

```go
func AssertMessageContent(t *testing.T, actual, expected string)
func AssertMessageCount(t *testing.T, messages []*pubsub.Message, expected int)
func AssertJSONEquals(t *testing.T, actual, expected []byte)
func AssertBinaryEquals(t *testing.T, actual, expected []byte)
func AssertContainsInOrder(t *testing.T, output string, expectedStrings []string)
```

### Benefits
- More descriptive test failures
- Consistent assertion patterns
- Better debugging information
- Reduced boilerplate in tests

## 6. Parallel Test Isolation

### Current Issue
Tests run in parallel but still have potential for interference through shared resources or timing.

### Recommended Approach
- Add test-specific prefixes to all resource names beyond current implementation
- Create a test context that includes isolation metadata
- Implement resource leak detection in cleanup
- Add mutex protection for any shared test utilities

### Benefits
- Better test isolation
- Ability to run more tests in parallel
- Easier debugging of test failures
- Protection against resource leaks

## 7. Create Command-Specific Test Helpers

### Current Issue
Similar command execution patterns repeated across tests.

### Recommended Approach
Create command-specific helpers:

```go
type DLRTestHelper struct {
    *BaseE2ETest
}

func (h *DLRTestHelper) RunWithActions(actions ...string) (string, error)
func (h *DLRTestHelper) VerifyMoveAction(messageContent string) error
func (h *DLRTestHelper) VerifyDiscardAction(messageContent string) error

type MoveTestHelper struct {
    *BaseE2ETest
}

func (h *MoveTestHelper) RunWithCount(count int) (string, error)
func (h *MoveTestHelper) VerifyAllMoved(expectedCount int) error
```

### Benefits
- Command-specific abstractions
- Clearer test intent
- Reduced duplication
- Easier to test command variations

## 8. Implement Test Scenarios as Data

### Current Issue
Each test scenario is implemented as code, making it harder to see the test cases at a glance.

### Recommended Approach
Define test scenarios as data structures:

```go
type TestScenario struct {
    Name           string
    Messages       []pubsub.Message
    UserInputs     []string
    ExpectedMoved  int
    ExpectedLeft   int
    ExpectedOutput []string
}

func RunDLRScenario(t *testing.T, scenario TestScenario)
```

### Benefits
- Table-driven tests
- Easy to add new scenarios
- Clear test documentation
- Reduced test code

## 9. Better Error Context

### Current Issue
When tests fail, it's not always clear what state the system was in.

### Recommended Approach
- Add context to all error messages
- Implement a test reporter that captures system state on failure
- Log important state transitions during test execution
- Create debug dumps for complex failures

### Benefits
- Faster debugging
- Better failure reports
- Easier to reproduce issues
- More maintainable tests

## 10. Configuration Management

### Current Issue
Test configuration (timeouts, retry counts) is hard-coded throughout tests.

### Recommended Approach
Create a test configuration system:

```go
type TestConfig struct {
    MessagePropagationTimeout time.Duration
    AckDeadlineTimeout       time.Duration
    DefaultRetryCount        int
    ParallelTests           int
}

func GetTestConfig() *TestConfig
```

### Benefits
- Centralized configuration
- Easy to adjust for different environments
- Better documentation of timing assumptions
- Ability to run tests in different modes

## 11. Test Organization

### Current Issue
All e2e tests are in a single package with many files.

### Recommended Approach
- Group related tests into sub-packages (e.g., `dlr_tests`, `move_tests`)
- Create a shared test utilities package
- Separate test data from test logic
- Consider using test suites for related tests

### Benefits
- Better code organization
- Easier navigation
- Clear test boundaries
- Improved maintainability

## 12. Resource Cleanup Verification

### Current Issue
Tests assume cleanup happens correctly but don't verify it.

### Recommended Approach
- Implement cleanup verification that runs after each test
- Check for orphaned topics/subscriptions
- Verify no messages left in test subscriptions
- Add resource leak detection

### Benefits
- Prevent test pollution
- Catch cleanup bugs early
- Better resource management
- More reliable test runs

## Implementation Priority

1. **High Priority** (Biggest impact, easiest to implement):
   - Replace hard-coded sleeps with smart polling (#1)
   - ~~Abstract stdin simulation (#3)~~ [COMPLETED]
   - Better assertion helpers (#5)

2. **Medium Priority** (Good value, moderate effort):
   - Create test suite base structure (#2)
   - ~~Improve test data management (#4)~~ [COMPLETED]
   - Create command-specific test helpers (#7)

3. **Lower Priority** (Nice to have, more effort):
   - Implement test scenarios as data (#8)
   - Test organization (#11)
   - Configuration management (#10)

## Next Steps

1. Start with high-priority items that provide immediate value
2. Implement changes incrementally to avoid breaking existing tests
3. Update tests one command at a time (start with either `dlr` or `move`)
4. Add new tests using the improved patterns
5. Gradually migrate existing tests to new patterns
6. Document the new testing patterns for future contributors

## Example Refactored Test

Here's how a test might look after applying these improvements:

```go
func TestDLRMoveAndDiscardScenario(t *testing.T) {
    scenario := TestScenario{
        Name: "Move one message and discard another",
        Messages: NewTestMessageBuilder().
            WithTextMessage("Message to move").
            WithTextMessage("Message to discard").
            Build(),
        UserInputs:     []string{"m", "d"},
        ExpectedMoved:  1,
        ExpectedLeft:   0,
        ExpectedOutput: []string{
            "moved successfully",
            "discarded (acked)",
        },
    }
    
    RunDLRScenario(t, scenario)
}
```

This refactored approach makes tests more readable, maintainable, and reliable while reducing code duplication and improving debugging capabilities.

