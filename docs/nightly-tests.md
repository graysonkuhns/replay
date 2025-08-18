# Nightly E2E Tests

## Overview

The nightly E2E tests run automatically at **2 AM UTC every day** via GitHub Actions. They provide comprehensive testing of the replay CLI tool with real GCP Pub/Sub resources.

## Workflow Details

- **Schedule**: `'0 2 * * *'` (2 AM UTC daily)
- **Workflow File**: `.github/workflows/nightly-e2e-tests.yml`
- **Manual Trigger**: Available via GitHub Actions UI "workflow_dispatch"

## Required Permissions

The workflow needs these GitHub permissions:
- `contents: read` - To checkout code
- `id-token: write` - For GCP authentication  
- `issues: write` - To create issues when tests fail

## Required Secrets

The workflow requires these repository secrets:
- `GCP_SA_KEY` - GCP service account credentials JSON
- `GCP_PROJECT_ID` - GCP project ID for testing

## Automatic Issue Creation

When nightly tests fail, the workflow automatically:
1. Creates a GitHub issue with the failure details
2. Tags it with `nightly-test-failure` and `bug` labels
3. Includes the run URL and run ID for investigation
4. Prevents duplicate issues for the same day

## Test Reliability Features

The e2e tests include several reliability features:

### Retry Mechanism
- `GetMessagesFromDestination()` retries up to 3 times with 5-second delays
- Handles GCP Pub/Sub message propagation timing issues
- Reduces false failures from network or timing issues

### Parallel Execution Control
- Nightly tests run with `PARALLEL_TESTS=1` for stability
- Regular PR tests can use higher parallelism for speed

### Resource Isolation
- Each test creates unique GCP resources with test run IDs
- Automatic cleanup after test completion
- No dependency on shared infrastructure

## Troubleshooting

### Common Issues

1. **"Resource not accessible by integration" (403 error)**
   - **Cause**: Missing `issues: write` permission
   - **Fix**: Ensure workflow has `issues: write` in permissions section

2. **"Expected X messages, got Y" errors**
   - **Cause**: GCP Pub/Sub message propagation delays
   - **Fix**: Tests now include retry mechanism (3 attempts, 5s delay)

3. **Authentication failures**
   - **Cause**: Invalid or missing GCP credentials
   - **Fix**: Update `GCP_SA_KEY` secret with valid service account JSON

4. **Test timeouts**
   - **Cause**: GCP resource creation delays or network issues
   - **Fix**: Tests have 60-minute timeout; check GCP project quotas

### Manual Testing

Use the provided test script to validate components locally:
```bash
export GCP_PROJECT=your-project-id
./scripts/test_nightly_workflow.sh
```

This checks:
- Workflow file syntax
- Go code compilation
- CLI functionality
- Test helper compilation

### Manual Workflow Trigger

To manually run nightly tests:
1. Go to [Nightly E2E Tests workflow](https://github.com/graysonkuhns/replay/actions/workflows/nightly-e2e-tests.yml)
2. Click "Run workflow" 
3. Select branch (usually `main`)
4. Click "Run workflow" to confirm

## Monitoring

Check workflow status:
- **GitHub Actions tab**: All workflow runs and their status
- **Issues tab**: Look for `nightly-test-failure` label for failures
- **Workflow artifacts**: Test results and logs (retained 30 days)

## Development Guidelines

### Adding New Tests
- Use `testhelpers.NewBaseE2ETest()` for consistent setup
- Include proper cleanup in test teardown
- Test with various message types (binary, JSON, plaintext)
- Consider parallel execution implications

### Improving Reliability
- Use the retry mechanisms in test helpers
- Add appropriate wait times for GCP resource propagation
- Include debugging logs for failure investigation
- Test edge cases that might cause intermittent failures

## Architecture

The nightly tests exercise:
- **CLI Commands**: `move` and `dlr` operations
- **Message Types**: Binary, JSON, plaintext, edge cases
- **GCP Integration**: Real Pub/Sub topics and subscriptions
- **User Interaction**: Simulated stdin for `dlr` command
- **Error Handling**: Invalid inputs, empty sources, etc.

This comprehensive testing ensures the replay CLI works correctly in real-world scenarios with actual GCP Pub/Sub infrastructure.