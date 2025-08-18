# Refactoring Implementation Tickets

This document contains individual, well-scoped tickets extracted from the refactoring opportunities documents. Tickets are organized by implementation phases, with clear indicators of which work can be done in parallel.

## Legend
- 🔵 Can be worked in parallel with other 🔵 tickets in the same phase
- 🔴 Must be completed sequentially (blocks other work)
- ⏱️ Estimated effort: S (Small: <1 day), M (Medium: 1-3 days), L (Large: 3-5 days)

---

## Phase 1: Foundation & Quick Wins
*These tickets establish foundations and deliver immediate value*

### 1.1 🔴 Extract Constants and Magic Values (Production) ⏱️ S
**Description**: Replace all hard-coded strings and values with named constants
- Create a `constants` package
- Define constants for broker types: "GCP_PUBSUB_SUBSCRIPTION", "GCP_PUBSUB_TOPIC"
- Define default timeouts and batch sizes
- Update all references throughout the codebase
**Blocks**: Ticket 2.3 (Config validation)

### 1.2 🔵 Complete Assertion Helpers (Test) ⏱️ M
**Description**: Implement remaining assertion helper functions
- Create `AssertMessageContent(t, actual, expected string)`
- Create `AssertMessageCount(t, messages, expected int)`
- Create `AssertJSONEquals(t, actual, expected []byte)`
- Create `AssertBinaryEquals(t, actual, expected []byte)`
- Create `AssertContainsInOrder(t, output, expectedStrings []string)`
- Add tests for all assertion helpers

### 1.3 🔵 Implement Structured Logging Interface (Production) ⏱️ M
**Description**: Create and implement structured logging
- Define Logger interface with Info, Error, Debug methods
- Create default implementation using existing log package
- Add context fields support (operation, messageID, etc.)
- Replace all fmt.Printf and log.Printf calls
- Ensure consistent log formatting

---

## Phase 2: Smart Polling & Error Handling
*Improve reliability and debugging capabilities*

### 2.1 🔴 Implement Smart Polling Helper (Test) ⏱️ L
**Description**: Replace hard-coded sleeps with intelligent polling
- Create `WaitForCondition` function with exponential backoff
- Create `WaitForMessagesInSubscription` specific helper
- Add timeout context support
- Replace all `time.Sleep()` calls in tests
- Add progress logging for long waits
**Blocks**: Better test reliability for all future test work

### 2.2 🔵 Define Custom Error Types (Production) ⏱️ S
**Description**: Create domain-specific error types
- Create `ValidationError` for config validation failures
- Create `BrokerError` for broker operation failures
- Create `TimeoutError` for polling timeouts
- Implement Error() methods with detailed context
- Update error returns throughout codebase

### 2.3 🔵 Enhance Configuration Validation (Production) ⏱️ S
**Description**: Add comprehensive config validation
- Implement `Validate()` method on CommandConfig
- Add resource name format validation
- Add project ID extraction validation
- Validate timeout ranges
- Return specific ValidationError instances
**Depends on**: Ticket 1.1 (Constants)

---

## Phase 3: Test Infrastructure
*Enhance test capabilities and organization*

### 3.1 🔵 Create Command-Specific Test Helpers (Test) ⏱️ M
**Description**: Build specialized test helpers for each command
- Create `DLRTestHelper` with RunWithActions, VerifyMoveAction, VerifyDiscardAction
- Create `MoveTestHelper` with RunWithCount, VerifyAllMoved
- Integrate with BaseE2ETest
- Add helper-specific assertion methods
- Update existing tests to use helpers

### 3.2 🔵 Implement Test Configuration System (Test) ⏱️ M
**Description**: Centralize test configuration
- Create TestConfig struct with timeouts, retry counts, etc.
- Support environment-based overrides
- Add configuration validation
- Create GetTestConfig() singleton
- Update all hardcoded test values

### 3.3 🔵 Add Resource Cleanup Verification (Test) ⏱️ M
**Description**: Ensure proper test cleanup
- Implement post-test verification hook
- Check for orphaned topics/subscriptions
- Verify no messages left in test subscriptions
- Add resource leak reporting
- Integrate with test teardown

---

## Phase 4: Dependency Injection & Testability
*Improve code testability and flexibility*

### 4.1 🔴 Implement Dependency Injection Framework (Production) ⏱️ L
**Description**: Add DI for better testability
- Create CommandFactory with injected dependencies
- Add Clock interface for time operations
- Add Reader interface for input operations
- Update command constructors to accept dependencies
- Create mock implementations for testing
**Blocks**: UI abstraction work

### 4.2 🔵 Add Unit Tests for Core Components (Production) ⏱️ L
**Description**: Add comprehensive unit tests
- Test MessageProcessor with mock broker
- Test DLRHandler with mock reader
- Test MoveHandler logic
- Test configuration parsing and validation
- Achieve >80% coverage on core packages

---

## Phase 5: Advanced Abstractions
*Enable future extensibility*

### 5.1 🔴 Implement BrokerFactory Pattern (Production) ⏱️ M
**Description**: Complete broker abstraction
- Create BrokerFactory interface
- Implement GCPPubSubBrokerFactory
- Add broker type registration
- Update broker creation logic
- Add factory tests
**Blocks**: Plugin architecture

### 5.2 🔵 Separate Interactive UI Logic (Production) ⏱️ L
**Description**: Abstract UI interactions
- Create UI interface with DisplayMessage, PromptAction, ShowProgress
- Implement TerminalUI
- Create JSONOutputUI for automation
- Update DLRHandler to use UI interface
- Add UI tests
**Depends on**: Ticket 4.1 (DI framework)

### 5.3 🔵 Implement Table-Driven Test Scenarios (Test) ⏱️ M
**Description**: Convert tests to data-driven format
- Define TestScenario struct
- Create RunDLRScenario helper
- Create RunMoveScenario helper
- Convert 3-5 existing tests as examples
- Document pattern for future tests

---

## Phase 6: Resilience & Performance
*Add production-grade reliability*

### 6.1 🔵 Add Retry Logic with Exponential Backoff (Production) ⏱️ M
**Description**: Implement retry patterns
- Create RetryConfig struct
- Implement WithRetry wrapper function
- Add exponential backoff calculation
- Apply to broker operations
- Add retry metrics/logging

### 6.2 🔵 Implement Circuit Breaker Pattern (Production) ⏱️ M
**Description**: Add circuit breaker for failing operations
- Create CircuitBreaker struct
- Implement state transitions (closed/open/half-open)
- Add failure threshold configuration
- Apply to broker publish operations
- Add circuit breaker metrics

### 6.3 🔵 Reorganize Test Package Structure (Test) ⏱️ S
**Description**: Better test organization
- Create e2e_tests/dlr/ subdirectory
- Create e2e_tests/move/ subdirectory
- Create e2e_tests/shared/ for common tests
- Move tests to appropriate packages
- Update imports

---

## Phase 7: Future Extensibility
*Long-term architectural improvements*

### 7.1 🔴 Design Plugin Architecture (Production) ⏱️ L
**Description**: Enable broker plugins
- Define BrokerPlugin interface
- Create PluginRegistry
- Add plugin discovery mechanism
- Create example plugin structure
- Document plugin development
**Depends on**: Ticket 5.1 (BrokerFactory)

### 7.2 🔵 Add Performance Benchmarks (Test) ⏱️ M
**Description**: Establish performance baselines
- Create benchmark tests for message processing
- Add memory usage benchmarks
- Benchmark broker operations
- Create performance regression detection
- Document performance targets

---

## Implementation Notes

### Parallel Work Opportunities
- Phase 1: Tickets 1.2 and 1.3 can be done simultaneously
- Phase 2: Tickets 2.2 and 2.3 can be done in parallel after 2.1
- Phase 3: All tickets (3.1, 3.2, 3.3) can be worked in parallel
- Phase 5: Tickets 5.2 and 5.3 can be done in parallel after dependencies
- Phase 6: All tickets (6.1, 6.2, 6.3) can be worked in parallel

### Critical Path
The critical path (sequential dependencies) is:
1. Extract Constants (1.1)
2. Smart Polling (2.1) 
3. Dependency Injection (4.1)
4. BrokerFactory (5.1)
5. Plugin Architecture (7.1)

### Quick Wins
For immediate impact, prioritize:
- Extract Constants (1.1) - Improves code clarity
- Complete Assertion Helpers (1.2) - Better test debugging
- Smart Polling (2.1) - Faster, more reliable tests

### Risk Mitigation
- Each ticket should include its own tests
- Major refactoring should preserve existing behavior
- Use feature flags for gradual rollout of new patterns
- Maintain backward compatibility for CLI interface
