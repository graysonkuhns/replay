package testhelpers

import (
	"testing"
	"time"
)

func TestTestContext(t *testing.T) {
	// Test creating a new test context
	testCtx := NewTestContext(t, "test_context_unit_test")

	// Verify initial state
	if testCtx.TestName != t.Name() {
		t.Errorf("Expected test name %s, got %s", t.Name(), testCtx.TestName)
	}

	if testCtx.TestID != "test_context_unit_test" {
		t.Errorf("Expected test ID 'test_context_unit_test', got %s", testCtx.TestID)
	}

	// Test resource naming
	topicName := testCtx.GenerateResourceName("topic", "events")
	if len(topicName) == 0 {
		t.Error("Generated resource name should not be empty")
	}

	// Verify the name contains expected components
	if !contains(topicName, "topic") || !contains(topicName, "events") {
		t.Errorf("Resource name %s should contain 'topic' and 'events'", topicName)
	}

	// Test attribute management
	testCtx.AddAttribute("custom_key", "custom_value")
	val, exists := testCtx.GetAttribute("custom_key")
	if !exists || val != "custom_value" {
		t.Errorf("Expected attribute 'custom_key' with value 'custom_value', got %s", val)
	}

	// Test resource tracking
	testCtx.TrackTopic("test-topic-1")
	testCtx.TrackSubscription("test-sub-1")
	testCtx.TrackTempFile("/tmp/test-file-1")
	testCtx.TrackConnection("conn-1")
	testCtx.TrackCustomResource("cache", "cache-1")

	// Get snapshot
	snapshot := testCtx.GetTrackedResources()

	// Verify tracked resources
	if len(snapshot.Topics) != 1 || snapshot.Topics[0] != "test-topic-1" {
		t.Errorf("Expected 1 topic 'test-topic-1', got %v", snapshot.Topics)
	}

	if len(snapshot.Subscriptions) != 1 || snapshot.Subscriptions[0] != "test-sub-1" {
		t.Errorf("Expected 1 subscription 'test-sub-1', got %v", snapshot.Subscriptions)
	}

	if len(snapshot.TempFiles) != 1 || snapshot.TempFiles[0] != "/tmp/test-file-1" {
		t.Errorf("Expected 1 temp file '/tmp/test-file-1', got %v", snapshot.TempFiles)
	}

	if len(snapshot.OpenConnections) != 1 || snapshot.OpenConnections[0] != "conn-1" {
		t.Errorf("Expected 1 connection 'conn-1', got %v", snapshot.OpenConnections)
	}

	if len(snapshot.CustomResources["cache"]) != 1 || snapshot.CustomResources["cache"][0] != "cache-1" {
		t.Errorf("Expected 1 cache resource 'cache-1', got %v", snapshot.CustomResources["cache"])
	}

	// Test leak detection
	if !snapshot.HasLeaks() {
		t.Error("Expected resources to be detected as leaks")
	}

	leakSummary := snapshot.GetLeakSummary()
	if !contains(leakSummary, "Resource leaks detected") {
		t.Errorf("Expected leak summary to contain 'Resource leaks detected', got: %s", leakSummary)
	}

	// Test resource untracking
	testCtx.UntrackTopic("test-topic-1")
	testCtx.UntrackSubscription("test-sub-1")
	testCtx.UntrackTempFile("/tmp/test-file-1")
	testCtx.UntrackConnection("conn-1")
	testCtx.UntrackCustomResource("cache", "cache-1")

	// Verify resources were untracked
	snapshot = testCtx.GetTrackedResources()
	if snapshot.HasLeaks() {
		t.Errorf("Expected no leaks after untracking, but found: %s", snapshot.GetLeakSummary())
	}
}

func TestParallelIndexUniqueness(t *testing.T) {
	// Test that parallel indices are unique across multiple test contexts
	indices := make(map[int32]bool)

	for i := 0; i < 10; i++ {
		testCtx := NewTestContext(t, "parallel_test")
		if indices[testCtx.ParallelIndex] {
			t.Errorf("Parallel index %d was already used", testCtx.ParallelIndex)
		}
		indices[testCtx.ParallelIndex] = true
	}

	if len(indices) != 10 {
		t.Errorf("Expected 10 unique parallel indices, got %d", len(indices))
	}
}

func TestResourceNameUniqueness(t *testing.T) {
	// Test that resource names are unique even for the same base name
	testCtx := NewTestContext(t, "uniqueness_test")

	name1 := testCtx.GenerateResourceName("topic", "events")

	// Sleep a tiny bit to ensure timestamp difference if using time-based uniqueness
	time.Sleep(time.Millisecond)

	// Create another context to simulate another test
	testCtx2 := NewTestContext(t, "uniqueness_test")
	name2 := testCtx2.GenerateResourceName("topic", "events")

	if name1 == name2 {
		t.Errorf("Expected different resource names for different test contexts, got same: %s", name1)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && s != substr && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 1; i < len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
