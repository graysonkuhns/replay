# GitHub Actions Integration Tests

This repository includes a GitHub Actions workflow that automatically runs integration tests for pull requests from trusted authors.

## Required Secrets

To enable the integration tests, the following GitHub repository secrets must be configured:

### `GCP_SA_KEY`
A service account key JSON for Google Cloud Platform authentication. This should be a base64-encoded service account key with permissions to:
- Create and manage Pub/Sub topics and subscriptions
- Access the test GCP project resources

### `GCP_PROJECT_ID`
The Google Cloud Platform project ID where the integration test resources will be created.

## How to Configure Secrets

1. Go to your repository's Settings → Secrets and variables → Actions
2. Click "New repository secret"
3. Add the two required secrets with their respective values

## Workflow Behavior

- **Trigger**: Runs on pull request events (opened, synchronize, reopened)
- **Author Check**: First job checks if the PR author is trusted
- **Integration Tests**: Second job runs only if the author is trusted
- **Timeout**: Tests have a 25-minute timeout (5 minutes more than the script's 20-minute timeout)
- **Artifacts**: Test logs are uploaded as artifacts for debugging

## Test Script

The workflow uses the existing `run_tests.sh` script which:
- Loads environment variables from `.env` if available
- Requires `GCP_PROJECT` environment variable
- Runs Go integration tests with a 20-minute timeout
- Supports running specific tests by name

## Manual Testing

To test the integration locally, ensure you have:
1. Go 1.23.0+ installed
2. `GCP_PROJECT` environment variable set
3. Google Cloud authentication configured
4. Run `./run_tests.sh`
