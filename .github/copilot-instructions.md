# Replay CLI Tool

Replay is a Go CLI tool for managing dead-lettered messages in GCP Pub/Sub. It provides commands to move messages and interactively review dead-lettered messages.

Always reference these instructions first and fallback to search or bash commands only when you encounter unexpected information that does not match the info here.

## Working Effectively

### Bootstrap and Build
- Download dependencies: `go mod download` (completes in ~2 seconds)
- Build the CLI: `go build .` (completes in ~1 second, creates `./replay` binary)
- The build is extremely fast - no special timeout needed

### Code Formatting and Quality
- Format code: `go fmt ./...`
- Run static analysis: `go vet ./...`
- Fix import formatting: `goimports -w .` (optional - may fix minor import grouping issues)
- All formatting commands complete in under 5 seconds

### Testing
- **CRITICAL**: This project has NO unit tests - only integration tests
- Run integration tests: `./run_tests.sh` 
- **NEVER CANCEL**: Integration tests take 45 minutes. Set timeout to 60+ minutes
- Integration tests require `GCP_PROJECT` environment variable set
- Tests create isolated GCP Pub/Sub resources and clean up automatically
- Parallel execution: Set `PARALLEL_TESTS=4` environment variable (default is 4)
- Run specific test: `./run_tests.sh TestName`

### Documentation
- Generate CLI docs: `./regenerate_docs.sh` (completes in ~1 second)
- Documentation is auto-generated from Cobra command definitions
- Generated docs are stored in `./docs/` directory

### CLI Usage Validation
- Test CLI help: `./replay --help`
- Test move command help: `./replay move --help`  
- Test dlr command help: `./replay dlr --help`
- All help commands should execute without errors and show proper usage

## Validation Scenarios

### Build Validation
Always run these commands to validate your changes:
1. `go mod download` - ensure dependencies are available
2. `go build .` - ensure code compiles successfully
3. `./replay --help` - ensure CLI starts and shows help
4. `go fmt ./...` - ensure code is properly formatted
5. `go vet ./...` - ensure no static analysis issues

### Integration Test Validation (Requires GCP Setup)
**NEVER CANCEL**: Tests take 45 minutes to complete. Always wait for completion.
1. Set `GCP_PROJECT` environment variable to your GCP project ID
2. Ensure `gcloud auth application-default login` is configured
3. Run `./run_tests.sh` and wait for completion
4. Tests will create and clean up their own Pub/Sub resources

### CLI Functionality Validation (Requires GCP Setup)
To test actual message processing functionality:
1. Set up GCP project with Pub/Sub topics and subscriptions
2. Create test messages using `./seed_test_dl_messages.sh` 
3. Test move command: `./replay move --source-type GCP_PUBSUB_SUBSCRIPTION --destination-type GCP_PUBSUB_TOPIC --source projects/PROJECT/subscriptions/NAME --destination projects/PROJECT/topics/NAME`
4. Test dlr command: `./replay dlr --source-type GCP_PUBSUB_SUBSCRIPTION --destination-type GCP_PUBSUB_TOPIC --source projects/PROJECT/subscriptions/NAME --destination projects/PROJECT/topics/NAME`

## Critical Information

### Build and Development Times
- **go build .**: 1 second
- **go fmt ./... && go vet ./...**: 5 seconds  
- **./regenerate_docs.sh**: 1 second
- **go mod download**: 2 seconds
- **./run_tests.sh**: 45 minutes - NEVER CANCEL

### Testing Requirements
- NO unit tests exist in this project
- Integration tests require GCP_PROJECT environment variable
- Integration tests create isolated resources automatically
- GitHub Actions runs tests with PARALLEL_TESTS=1 for CI stability

### Key Commands
The CLI provides two main commands:
- `move`: Moves all messages from source to destination sequentially
- `dlr`: Interactive dead letter review - allows user to choose move/discard for each message

### Code Structure
- `main.go`: Entry point
- `cmd/`: Cobra command definitions (root.go, move.go, dlr.go)
- `integration_tests/`: All tests are integration tests requiring GCP
- `tools/docgen/`: Documentation generation utility
- `test_environment/`: Terraform configuration for shared test resources (optional)

## Common Tasks

### Repository Root Structure
```
.
├── README.md
├── go.mod
├── go.sum  
├── main.go
├── cmd/
├── integration_tests/
├── docs/
├── tools/
├── scripts/
├── run_tests.sh
├── regenerate_docs.sh
└── test_environment/
```

### Example Build Sequence
1. `go mod download`
2. `go build .`
3. `./replay --help` # Validate CLI works
4. `go fmt ./... && go vet ./...` # Format and lint
5. `./regenerate_docs.sh` # Update documentation if CLI changed
6. For full validation: `GCP_PROJECT=your-project ./run_tests.sh` (45+ minutes)

### Cleanup Scripts
- `./scripts/delete_pubsub_topics.sh` - Delete all Pub/Sub topics in project
- `./scripts/delete_pubsub_subscriptions.sh` - Delete all Pub/Sub subscriptions in project
- Use with caution - these delete ALL resources of the specified type

## Important Notes
- Always ensure your changes work with the fast build cycle before running integration tests
- Integration tests are the primary validation method - there are no unit tests
- The application requires GCP authentication via `gcloud auth application-default login`
- Both move and dlr commands require valid GCP Pub/Sub resource names
- Use `--pretty-json` flag with dlr command for formatted JSON message display