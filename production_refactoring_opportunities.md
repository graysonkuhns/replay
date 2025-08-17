# Production Code Refactoring Opportunities

This document outlines refactoring opportunities identified in the production codebase of the Replay CLI tool. The analysis focuses on improving code maintainability, testability, and extensibility.

## Executive Summary

The Replay CLI tool has significant opportunities for refactoring, primarily centered around eliminating code duplication between the `dlr` and `move` commands, introducing proper abstractions, and separating concerns. The current implementation has ~80% code duplication between the two main commands and tightly couples business logic with CLI framework code.

## Major Refactoring Opportunities

### 1. Eliminate Massive Code Duplication

**Current State:**
- `cmd/dlr.go` and `cmd/move.go` share approximately 80% of their code
- Both files implement identical:
  - Flag parsing logic
  - Resource validation
  - Project ID extraction from resource names
  - Pub/Sub client creation
  - Message polling loops (with minor variations)

**Proposed Refactoring:**
```go
// Create a shared message broker abstraction
type MessageBroker interface {
    Pull(ctx context.Context, config PullConfig) (*Message, error)
    Publish(ctx context.Context, message *Message) error
    Acknowledge(ctx context.Context, ackID string) error
}

// Create a shared command configuration
type CommandConfig struct {
    SourceType      string
    DestinationType string
    Source          string
    Destination     string
    Count           int
    PollTimeout     time.Duration
}

// Extract shared logic into a service layer
type MessageProcessor struct {
    broker MessageBroker
    config CommandConfig
}
```

### 2. Extract Business Logic from Command Handlers

**Current State:**
- The `Run` functions in both commands are 100+ lines long
- Business logic is mixed with:
  - Flag parsing
  - Validation
  - Client initialization
  - Error handling
  - Logging

**Proposed Refactoring:**
```go
// Separate concerns into distinct layers
type MessageService interface {
    ProcessMessages(ctx context.Context, handler MessageHandler) error
}

type MessageHandler interface {
    HandleMessage(msg *Message) (Action, error)
}

// Interactive handler for dlr command
type InteractiveHandler struct {
    reader *bufio.Reader
}

// Automatic handler for move command
type AutomaticHandler struct{}
```

### 3. Introduce Proper Abstraction for Message Brokers

**Current State:**
- Direct use of Google Pub/Sub APIs throughout the code
- Hard-coded support for only GCP Pub/Sub
- Difficult to add support for other message brokers

**Proposed Refactoring:**
```go
// Define broker-agnostic interfaces
type BrokerFactory interface {
    CreateBroker(brokerType string, config map[string]string) (MessageBroker, error)
}

// Implement specific brokers
type GCPPubSubBroker struct {
    subClient   *pubsub.Client
    topicClient *pubsub.Client
    // ... other fields
}

// Future implementations
type AWSKinesisBroker struct{}
type KafkaBroker struct{}
```

### 4. Improve Configuration Management

**Current State:**
- Flags are parsed individually in each command
- No configuration validation beyond basic type checks
- Resource name parsing is done with string splitting

**Proposed Refactoring:**
```go
// Create proper configuration structures
type Config struct {
    Source      ResourceConfig
    Destination ResourceConfig
    Processing  ProcessingConfig
}

type ResourceConfig struct {
    Type     BrokerType
    Resource Resource
}

type Resource struct {
    Project      string
    Subscription string // or Topic
}

// Add configuration validation
func (c *Config) Validate() error {
    // Comprehensive validation logic
}

// Add resource parsing
func ParseResource(resourceName string) (*Resource, error) {
    // Robust parsing with proper error handling
}
```

### 5. Standardize Error Handling and Logging

**Current State:**
- Inconsistent error handling patterns
- `dlr` uses `fmt.Printf` for output
- `move` uses `log.Printf` for output
- No structured logging

**Proposed Refactoring:**
```go
// Introduce structured logging
type Logger interface {
    Info(msg string, fields ...Field)
    Error(msg string, err error, fields ...Field)
    Debug(msg string, fields ...Field)
}

// Define custom errors
type ValidationError struct {
    Field   string
    Message string
}

type BrokerError struct {
    Operation string
    Cause     error
}
```

### 6. Extract Constants and Magic Values

**Current State:**
- Hard-coded strings: "GCP_PUBSUB_SUBSCRIPTION", "GCP_PUBSUB_TOPIC"
- Magic numbers: polling timeout defaults
- Resource format assumptions

**Proposed Refactoring:**
```go
// Define constants in a dedicated package
package constants

const (
    BrokerTypeGCPPubSubSubscription = "GCP_PUBSUB_SUBSCRIPTION"
    BrokerTypeGCPPubSubTopic       = "GCP_PUBSUB_TOPIC"
    
    DefaultPollTimeout = 10 * time.Second
    DefaultBatchSize   = 1
)

// Define supported broker types
var SupportedSourceTypes = []string{
    BrokerTypeGCPPubSubSubscription,
}

var SupportedDestinationTypes = []string{
    BrokerTypeGCPPubSubTopic,
}
```

### 7. Improve Testability

**Current State:**
- Commands are difficult to test due to tight coupling
- No dependency injection
- Direct creation of clients within command handlers

**Proposed Refactoring:**
```go
// Use dependency injection
type CommandFactory struct {
    brokerFactory BrokerFactory
    logger        Logger
}

func (f *CommandFactory) CreateDLRCommand() *cobra.Command {
    return &cobra.Command{
        Run: func(cmd *cobra.Command, args []string) {
            // Use injected dependencies
        },
    }
}

// Make components testable with interfaces
type Clock interface {
    Now() time.Time
}

type Reader interface {
    ReadString(delim byte) (string, error)
}
```

### 8. Separate Interactive UI Logic

**Current State:**
- User interaction logic is embedded in the dlr command
- No separation between UI and business logic

**Proposed Refactoring:**
```go
// Create a UI abstraction
type UI interface {
    DisplayMessage(msg *Message) error
    PromptAction() (Action, error)
    ShowProgress(processed, total int)
}

type TerminalUI struct {
    reader *bufio.Reader
    writer io.Writer
}

// Allow for different UI implementations
type JSONOutputUI struct{}
type WebUI struct{}
```

### 9. Add Retry and Circuit Breaker Patterns

**Current State:**
- Basic error handling with continue statements
- No retry logic for transient failures
- No circuit breaker for failing brokers

**Proposed Refactoring:**
```go
// Add resilience patterns
type RetryConfig struct {
    MaxAttempts int
    BackoffBase time.Duration
}

type CircuitBreaker struct {
    maxFailures int
    timeout     time.Duration
}

func WithRetry(fn func() error, config RetryConfig) error {
    // Implement exponential backoff
}
```

### 10. Create a Plugin Architecture

**Current State:**
- Hard-coded support for only GCP Pub/Sub
- Adding new brokers requires modifying core code

**Proposed Refactoring:**
```go
// Define plugin interface
type BrokerPlugin interface {
    Name() string
    SupportedSourceTypes() []string
    SupportedDestinationTypes() []string
    CreateBroker(config map[string]string) (MessageBroker, error)
}

// Plugin registry
type PluginRegistry struct {
    plugins map[string]BrokerPlugin
}

func (r *PluginRegistry) Register(plugin BrokerPlugin) {
    r.plugins[plugin.Name()] = plugin
}
```

## Implementation Priority

1. **High Priority** (Immediate benefits, low risk):
   - Extract shared code between dlr and move commands
   - Create configuration structures
   - Standardize error handling and logging

2. **Medium Priority** (Significant benefits, moderate effort):
   - Introduce message broker abstraction
   - Separate business logic from command handlers
   - Extract constants and magic values

3. **Low Priority** (Future extensibility):
   - Plugin architecture
   - Retry and circuit breaker patterns
   - Alternative UI implementations

## Migration Strategy

1. **Phase 1**: Extract shared code into internal packages without changing external behavior
2. **Phase 2**: Introduce abstractions and interfaces
3. **Phase 3**: Refactor commands to use new abstractions
4. **Phase 4**: Add tests for new components
5. **Phase 5**: Implement advanced features (plugins, resilience patterns)

## Benefits of Refactoring

1. **Reduced Maintenance**: Eliminating duplication reduces the surface area for bugs
2. **Improved Testability**: Abstractions and dependency injection enable comprehensive unit testing
3. **Better Extensibility**: Adding new message brokers becomes trivial
4. **Enhanced Reliability**: Proper error handling and resilience patterns improve stability
5. **Cleaner Architecture**: Separation of concerns makes the codebase easier to understand

## Risks and Mitigation

1. **Risk**: Breaking existing functionality
   - **Mitigation**: Implement changes incrementally with comprehensive integration tests

2. **Risk**: Over-engineering
   - **Mitigation**: Focus on immediate pain points first, defer advanced features

3. **Risk**: Performance regression
   - **Mitigation**: Benchmark critical paths before and after refactoring

## Conclusion

The Replay CLI tool has significant technical debt in the form of code duplication and tight coupling. The proposed refactoring would transform it into a maintainable, extensible, and testable codebase. The highest priority should be eliminating the massive duplication between the dlr and move commands, which alone would reduce the codebase by approximately 40% while improving maintainability.
