# Test Environment

## Prerequisites

* [tfenv](https://github.com/tfutils/tfenv)
* A GCP project to deploy test resources in

## Setup

* Allow terraform to access your GCP project - `gcloud auth application-default login`
* Install terraform - `tfenv install` (version from .terraform-version file will be used)
* Create terraform variables file - `cp terraform.tfvars.template terraform.tfvars`
* Update values in `terraform.tfvars` file
* Install terraform modules - `terraform init`

## Integration Tests

**Note:** As of the latest version, integration tests now create their own fresh GCP Pub/Sub resources (topics and subscriptions) for each test run. This means:

- Each test gets isolated resources with unique names
- Tests no longer depend on pre-existing Terraform-managed resources
- Tests automatically clean up their resources after completion
- You only need to ensure the `GCP_PROJECT` environment variable is set

### Parallel Execution

Integration tests now run in parallel by default, significantly reducing execution time. The `run_tests.sh` script supports configurable parallelism:

- Default: 4 concurrent tests
- Configurable via `PARALLEL_TESTS` environment variable
- Example: `PARALLEL_TESTS=8 ./run_tests.sh` to run 8 tests concurrently

The Terraform configuration in this directory can still be used to create a shared test environment if needed for manual testing or other purposes.
