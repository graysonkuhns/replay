# GitHub Actions Workflows

This directory contains the CI/CD workflows for the Replay CLI project.

## Workflows

### 1. E2E Tests (`e2e-tests.yml`)
Runs on every pull request to ensure code quality. The tests are distributed across 3 parallel workers to improve execution time.

**Key Features:**
- Runs tests across 3 GitHub Actions workers concurrently
- Each worker runs its assigned tests with parallelism of 10
- Uploads test artifacts separately for each worker
- Includes a summary job to check overall test status

### 2. Nightly E2E Tests (`nightly-e2e-tests.yml`)
Runs every night at 2 AM UTC to catch regressions and flaky tests.

**Key Features:**
- Same parallel execution strategy as PR tests (3 workers)
- Automatically creates GitHub issues when tests fail
- Consolidates failure reports from all workers
- Retains test artifacts for 30 days

### 3. Lint (`lint.yml`)
Runs Go linting checks on pull requests.

## Parallel Test Execution

The E2E tests use a parallel execution strategy to reduce overall test runtime:

1. **Test Discovery**: Uses `tools/discover_tests.go` to find all test functions
2. **Test Distribution**: Uses `tools/distribute_tests.go` to evenly distribute tests across workers
3. **Worker Execution**: Each worker runs approximately 1/3 of the tests

### Configuration
- **Number of Workers**: 3 (configurable in workflow files)
- **Parallelism per Worker**: 10 (set via `PARALLEL_TESTS` environment variable)

### How It Works

1. The workflow creates a matrix strategy with 3 workers (0, 1, 2)
2. Each worker:
   - Discovers all available tests
   - Calculates which tests it should run based on its index
   - Runs only its assigned tests with parallelism of 10
3. A summary job checks if all workers completed successfully

### Test Distribution Example
With 100 tests and 3 workers:
- Worker 0: Tests 1-34 (34 tests)
- Worker 1: Tests 35-67 (33 tests)  
- Worker 2: Tests 68-100 (33 tests)

### Running Tests Locally
You can use the same distribution mechanism locally:

```bash
# Run as worker 0 of 3
WORKER_INDEX=0 TOTAL_WORKERS=3 PARALLEL_TESTS=10 ./run_e2e_tests_subset.sh

# Run all tests (no distribution)
PARALLEL_TESTS=10 ./run_tests.sh
```

## Modifying the Workflows

### Changing Number of Workers
To change the number of parallel workers:

1. Update the matrix in the workflow file:
   ```yaml
   strategy:
     matrix:
       worker: [0, 1, 2, 3, 4]  # For 5 workers
   ```

2. Update the `TOTAL_WORKERS` environment variable:
   ```yaml
   env:
     TOTAL_WORKERS: 5
   ```

3. For nightly tests, also update the worker loop in the summary job

### Changing Parallelism per Worker
Modify the `PARALLEL_TESTS` environment variable in the run step:
```yaml
PARALLEL_TESTS=20 WORKER_INDEX=${{ matrix.worker }} TOTAL_WORKERS=3 ./run_e2e_tests_subset.sh
```

## Troubleshooting

### Tests Not Running on a Worker
- Check the test discovery output in the workflow logs
- Ensure tests follow the naming convention `Test*` and are in `*_test.go` files
- Verify the test takes `*testing.T` as a parameter

### Uneven Test Distribution
The distribution algorithm ensures the difference between workers is at most 1 test. If you see larger differences, check:
- Test discovery is finding all tests
- No tests are being filtered out incorrectly

### Worker Failures
- Each worker uploads its own artifacts
- Check the specific worker's logs and artifacts
- The summary job will indicate which workers failed