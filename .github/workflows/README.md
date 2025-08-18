# GitHub Actions Integration Tests

This repository includes GitHub Actions workflows that automatically run e2e tests:
1. **PR Integration Tests** - Run on pull requests from trusted authors
2. **Nightly Integration Tests** - Run automatically every night at 2 AM UTC

## Required Secrets

To enable the e2e tests, the following GitHub repository secrets must be configured:

### `GCP_SA_KEY`
A service account key JSON for Google Cloud Platform authentication. This should be a base64-encoded service account key with permissions to:
- Create and manage Pub/Sub topics and subscriptions
- Access the test GCP project resources

### `GCP_PROJECT_ID`
The Google Cloud Platform project ID where the e2e test resources will be created.

## How to Configure Secrets

1. Go to your repository's Settings → Secrets and variables → Actions
2. Click "New repository secret"
3. Add the two required secrets with their respective values

## Workflow Behavior

### PR Integration Tests
- **Trigger**: Runs on pull request events (opened, synchronize, reopened)
- **Parallelism**: Limited to 1 concurrent test for resource efficiency
- **Integration Tests**: Run the e2e tests
- **Timeout**: Tests have a 50-minute timeout
- **Artifacts**: Test logs are uploaded as artifacts for 7 days

### Nightly Integration Tests
- **Trigger**: Runs automatically every night at 2 AM UTC
- **Manual Trigger**: Can also be triggered manually via workflow_dispatch
- **Parallelism**: Uses default parallelism (4 concurrent tests) for better performance
- **Integration Tests**: Run all e2e tests
- **Timeout**: Tests have a 50-minute timeout
- **Artifacts**: Test logs are uploaded as artifacts for 30 days
- **Failure Handling**: Automatically creates GitHub issues when tests fail

## Test Script

The workflow uses the existing `run_tests.sh` script which:
- Loads environment variables from `.env` if available
- Requires `GCP_PROJECT` environment variable
- Runs Go e2e tests with a 20-minute timeout
- Supports running specific tests by name

## Manual Testing

To test the integration locally, ensure you have:
1. Go 1.23.0+ installed
2. `GCP_PROJECT` environment variable set
3. Google Cloud authentication configured
4. Run `./run_tests.sh`
