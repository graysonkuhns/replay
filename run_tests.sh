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

echo "Running all tests..."
go test -v ./...
