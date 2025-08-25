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
PARALLEL_TESTS=${PARALLEL_TESTS:-10}

# Build the replay binary
echo "Building replay binary..."
go build -o replay .
if [ $? -ne 0 ]; then
  echo "Failed to build replay binary"
  exit 1
fi

# Export workspace root for tests to find the binary
export REPLAY_WORKSPACE_ROOT=$(pwd)

# Cleanup function to remove the binary
cleanup() {
  echo "Cleaning up..."
  rm -f replay
}

# Set up trap to cleanup on exit
trap cleanup EXIT

# Check if we're running a subset of tests based on worker configuration
if [ -n "$WORKER_INDEX" ] && [ -n "$TOTAL_WORKERS" ]; then
  echo "Running as worker $WORKER_INDEX of $TOTAL_WORKERS"
  
  # Build the test discovery tool
  echo "Building test discovery tool..."
  go build -o discover_tests ./tools/discover_tests
  go build -o distribute_tests ./tools/distribute_tests
  
  # Discover all tests and distribute to this worker
  echo "Discovering tests..."
  ALL_TESTS=$(./discover_tests ./e2e_tests)
  TOTAL_TEST_COUNT=$(echo "$ALL_TESTS" | wc -l)
  echo "Total tests found: $TOTAL_TEST_COUNT"
  
  # Get tests for this worker
  WORKER_TESTS=$(echo "$ALL_TESTS" | ./distribute_tests $WORKER_INDEX $TOTAL_WORKERS)
  WORKER_TEST_COUNT=$(echo "$WORKER_TESTS" | wc -l)
  echo "Worker $WORKER_INDEX will run $WORKER_TEST_COUNT tests"
  
  # Clean up discovery tools
  rm -f discover_tests distribute_tests
  
  # Build regex pattern for tests
  if [ -n "$WORKER_TESTS" ]; then
    # Convert test list to regex pattern
    TEST_PATTERN=$(echo "$WORKER_TESTS" | paste -sd '|' -)
    echo "Running tests matching pattern: $TEST_PATTERN"
    go test -count 1 -v -timeout 45m -parallel ${PARALLEL_TESTS} ./e2e_tests -run "^($TEST_PATTERN)$"
  else
    echo "No tests assigned to this worker"
    exit 0
  fi
elif [ $# -eq 0 ]; then
  # Run all tests
  echo "Running all tests in parallel (max ${PARALLEL_TESTS} concurrent tests)..."
  go test -count 1 -v -timeout 45m -parallel ${PARALLEL_TESTS} ./e2e_tests
else
  # Run only the specified test
  TEST_NAME=$1
  echo "Running test: $TEST_NAME"
  go test -count 1 -v -timeout 45m -parallel ${PARALLEL_TESTS} ./e2e_tests -run "$TEST_NAME"
fi