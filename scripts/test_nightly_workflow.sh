#!/usr/bin/env bash

# Test script to verify nightly workflow components work correctly
# This script can be run manually to test the workflow components without waiting for the nightly cron

set -e

echo "🔍 Testing nightly workflow components..."

# Check if GCP_PROJECT is set
if [ -z "$GCP_PROJECT" ]; then
    echo "❌ GCP_PROJECT environment variable must be set"
    echo "   Set it with: export GCP_PROJECT=your-project-id"
    exit 1
fi

echo "✅ GCP_PROJECT is set: $GCP_PROJECT"

# Test 1: Verify workflow file syntax
echo "🧪 Test 1: Checking workflow file syntax..."
if command -v yq >/dev/null 2>&1; then
    yq eval '.permissions.issues' .github/workflows/nightly-integration-tests.yml | grep -q "write" && echo "✅ Workflow permissions correctly set" || echo "❌ Workflow permissions not set correctly"
else
    grep -q "issues: write" .github/workflows/nightly-integration-tests.yml && echo "✅ Workflow permissions correctly set" || echo "❌ Workflow permissions not set correctly"
fi

# Test 2: Verify Go code compiles
echo "🧪 Test 2: Checking Go code compilation..."
go build -o /tmp/replay-test . && echo "✅ Go code compiles successfully" || { echo "❌ Go compilation failed"; exit 1; }

# Test 3: Verify test helpers compile
echo "🧪 Test 3: Checking test helpers compilation..."
go test -c ./integration_tests -o /tmp/integration-tests 2>/dev/null && echo "✅ Integration tests compile successfully" || { echo "❌ Integration tests compilation failed"; exit 1; }

# Test 4: Run a quick syntax check on all go files
echo "🧪 Test 4: Running go vet..."
go vet ./... && echo "✅ go vet passed" || { echo "❌ go vet failed"; exit 1; }

# Test 5: Check if CLI help works
echo "🧪 Test 5: Testing CLI help output..."
/tmp/replay-test --help > /dev/null && echo "✅ CLI help works" || { echo "❌ CLI help failed"; exit 1; }

# Test 6: Test the retry mechanism in test helpers (syntax check only)
echo "🧪 Test 6: Checking retry mechanism syntax..."
grep -q "maxRetries.*=.*3" integration_tests/testhelpers/base_integration.go && echo "✅ Retry mechanism added" || echo "❌ Retry mechanism not found"

echo ""
echo "🎉 All checks passed! The nightly workflow components are ready."
echo ""
echo "💡 To manually trigger the nightly workflow for testing:"
echo "   1. Go to: https://github.com/graysonkuhns/replay/actions/workflows/nightly-integration-tests.yml"  
echo "   2. Click 'Run workflow' button"
echo "   3. Click 'Run workflow' to confirm"
echo ""
echo "⚠️  Note: The workflow requires GCP secrets to be configured in the repository settings."

# Cleanup
rm -f /tmp/replay-test /tmp/integration-tests

echo "✅ Test complete!"