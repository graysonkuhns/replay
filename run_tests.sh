#!/usr/bin/env bash
set -e

# Load environment variables from .env if available
if [ -f .env ]; then
  set -a
  source .env
  set +a
fi

if [ -z "$GCP_PROJECT" ]; then
  echo "GCP_PROJECT environment variable must be set"
  exit 1
fi

# Set default parallelism (can be overridden via PARALLEL_TESTS environment variable)
PARALLEL_TESTS=${PARALLEL_TESTS:-4}

# Check if a specific test name was provided
if [ $# -eq 0 ]; then
  echo "Running all tests in parallel (max ${PARALLEL_TESTS} concurrent tests)..."
  go test -count 1 -v -timeout 20m -parallel ${PARALLEL_TESTS} ./integration_tests
else
  # Run only the specified test
  TEST_NAME=$1
  echo "Running test: $TEST_NAME"
  go test -count 1 -v -timeout 20m -parallel ${PARALLEL_TESTS} ./integration_tests -run "$TEST_NAME"
fi
