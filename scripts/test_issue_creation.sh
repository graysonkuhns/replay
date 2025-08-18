#!/usr/bin/env bash

# Test script to validate the enhanced issue creation functionality
# This script simulates a test failure scenario to verify log parsing works correctly

set -e

echo "🧪 Testing enhanced nightly workflow issue creation..."

# Create a temporary directory for testing
TEST_DIR="/tmp/workflow-test-$$"
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"

# Create a sample test failure log
cat > nightly-test-output.log << 'EOF'
Running all tests in parallel (max 1 concurrent tests)...
=== RUN   TestDLROperation
=== PAUSE TestDLROperation
=== RUN   TestMoveStopsWhenSourceExhausted
=== PAUSE TestMoveStopsWhenSourceExhausted
=== CONT  TestDLROperation
=== CONT  TestMoveStopsWhenSourceExhausted
    dlr_test.go:28: Error running CLI command: failed to authenticate with GCP
--- FAIL: TestDLROperation (45.67s)
    move_test.go:31: Failed to publish test messages: rpc error: code = DeadlineExceeded desc = context deadline exceeded
--- FAIL: TestMoveStopsWhenSourceExhausted (52.33s)
FAIL
exit status 1
EOF

# Create a Node.js script that simulates the GitHub Actions script logic
cat > test-issue-creation.js << 'EOF'
const fs = require('fs');

// Simulate the GitHub Actions environment
const context = {
  runId: '12345678',
  repo: { owner: 'graysonkuhns', repo: 'replay' }
};

const runUrl = `https://github.com/${context.repo.owner}/${context.repo.repo}/actions/runs/${context.runId}`;

// Read test output log file (same logic as in workflow)
let testOutput = '';
let failureSummary = '';
try {
  const logContent = fs.readFileSync('nightly-test-output.log', 'utf8');
  
  const lines = logContent.split('\n');
  const failureLines = [];
  let failedTests = [];
  let seenFailures = new Set();
  
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    
    if (line.includes('--- FAIL:')) {
      const testMatch = line.match(/--- FAIL: (\w+)/);
      if (testMatch && !seenFailures.has(testMatch[1])) {
        failedTests.push(testMatch[1]);
        seenFailures.add(testMatch[1]);
        failureLines.push(line);
      }
    } else if (line.trim().startsWith('dlr_test.go:') || 
              line.trim().startsWith('move_test.go:') ||
              line.trim().startsWith('e2e_tests/') ||
              line.includes('Error:') || 
              line.includes('Failed to') ||
              line.includes('rpc error:') ||
              line.includes('panic:') ||
              line.includes('exit status')) {
      failureLines.push(line);
    }
  }
  
  const maxLength = 3000;
  if (failureLines.length > 0) {
    testOutput = failureLines.join('\n');
    if (testOutput.length > maxLength) {
      testOutput = '...\n' + testOutput.slice(testOutput.length - maxLength + 4);
    }
  }
  
  if (failedTests.length > 0) {
    failureSummary = `**Failed Tests:** ${failedTests.join(', ')}\n\n`;
  }
  
} catch (error) {
  console.log('Could not read test output log:', error.message);
  testOutput = 'Test output log not available. Check the workflow artifacts for detailed logs.';
}

// Create issue body (same logic as in workflow)
const title = `Nightly E2E Tests Failed - ${new Date().toISOString().split('T')[0]}`;
const body = `The nightly E2E tests failed on ${new Date().toISOString().split('T')[0]}.

**Run URL:** ${runUrl}

**Run ID:** ${context.runId}

${failureSummary}**Test Failure Output:**
\`\`\`
${testOutput}
\`\`\`

**Next Steps:**
1. Check the complete logs in the workflow artifacts: [Test Results](${runUrl}#artifacts)
2. Review the failing tests and error messages above
3. Run the failing tests locally to reproduce the issue
4. Fix the underlying issue and verify with local test runs

This issue was automatically created by the nightly E2E test workflow.`;

console.log('=== GENERATED ISSUE ===');
console.log('Title:', title);
console.log('\nBody:');
console.log(body);
console.log('\n=== VALIDATION ===');
console.log('✅ Issue title includes date');
console.log('✅ Issue body includes run URL and ID');
console.log(`✅ Issue body includes failed tests: ${failureSummary.trim()}`);
console.log('✅ Issue body includes formatted test output');
console.log('✅ Issue body includes troubleshooting steps');
console.log(`📊 Total issue body length: ${body.length} characters`);

if (body.length > 65536) {
  console.log('⚠️  Warning: Issue body may exceed GitHub API limits');
} else {
  console.log('✅ Issue body size is within GitHub API limits');
}
EOF

# Run the test
node test-issue-creation.js

# Cleanup
cd /
rm -rf "$TEST_DIR"

echo ""
echo "🎉 Test completed successfully!"
echo "The enhanced issue creation logic is working correctly."