package testhelpers

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestContext provides comprehensive test isolation metadata and tracking
type TestContext struct {
	// Embedded context for cancellation and deadlines
	context.Context

	// Test metadata
	TestName      string
	TestID        string // Unique identifier for this test instance
	StartTime     time.Time
	ParallelIndex int32 // Index for tests running in parallel

	// Resource tracking
	ResourcePrefix string               // Prefix for all resources created by this test
	Resources      *TestResourceTracker // Tracks created resources for leak detection

	// Test-specific attributes
	Attributes map[string]string // Custom attributes for test filtering/tagging
	mu         sync.RWMutex      // Protects attributes map
}

// TestResourceTracker tracks resources created during test execution
type TestResourceTracker struct {
	mu              sync.Mutex
	Topics          []string
	Subscriptions   []string
	TempFiles       []string
	OpenConnections []string
	CustomResources map[string][]string // For extensibility
}

// Global counter for generating unique parallel indices
var parallelIndexCounter int32

// sanitizeTestName cleans a test name for use in resource names
func sanitizeTestName(name string) string {
	// Replace common problematic characters
	replacer := strings.NewReplacer(
		"/", "_",
		" ", "_",
		".", "_",
		"-", "_",
		"(", "",
		")", "",
		"[", "",
		"]", "",
	)

	sanitized := replacer.Replace(name)
	// Convert to lowercase and trim to reasonable length
	sanitized = strings.ToLower(sanitized)
	if len(sanitized) > 50 {
		sanitized = sanitized[:50]
	}

	return sanitized
}

// NewTestContext creates a new test context with isolation metadata
func NewTestContext(t *testing.T, testRunID string) *TestContext {
	// Generate a unique parallel index for this test
	parallelIdx := atomic.AddInt32(&parallelIndexCounter, 1)

	// Create a unique resource prefix combining test info and parallel index
	// Include nanosecond timestamp and random component for maximum uniqueness
	resourcePrefix := fmt.Sprintf("test_%s_%d_%d_%d",
		sanitizeTestName(t.Name()),
		time.Now().UnixNano(),
		parallelIdx,
		rand.Intn(10000))

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
	})

	tc := &TestContext{
		Context:        ctx,
		TestName:       t.Name(),
		TestID:         testRunID,
		StartTime:      time.Now(),
		ParallelIndex:  parallelIdx,
		ResourcePrefix: resourcePrefix,
		Resources: &TestResourceTracker{
			CustomResources: make(map[string][]string),
		},
		Attributes: map[string]string{
			"testRun":       testRunID,
			"testName":      t.Name(),
			"parallelIndex": fmt.Sprintf("%d", parallelIdx),
		},
	}

	return tc
}

// AddAttribute adds a custom attribute to the test context
func (tc *TestContext) AddAttribute(key, value string) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.Attributes[key] = value
}

// GetAttribute retrieves a custom attribute from the test context
func (tc *TestContext) GetAttribute(key string) (string, bool) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	val, exists := tc.Attributes[key]
	return val, exists
}

// GetAllAttributes returns a copy of all attributes
func (tc *TestContext) GetAllAttributes() map[string]string {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	attrs := make(map[string]string, len(tc.Attributes))
	for k, v := range tc.Attributes {
		attrs[k] = v
	}
	return attrs
}

// GenerateResourceName generates a test-specific resource name with proper isolation
func (tc *TestContext) GenerateResourceName(resourceType, baseName string) string {
	return fmt.Sprintf("%s_%s_%s", tc.ResourcePrefix, resourceType, baseName)
}

// TrackTopic records a topic created during the test
func (tc *TestContext) TrackTopic(topicName string) {
	tc.Resources.mu.Lock()
	defer tc.Resources.mu.Unlock()
	tc.Resources.Topics = append(tc.Resources.Topics, topicName)
}

// TrackSubscription records a subscription created during the test
func (tc *TestContext) TrackSubscription(subName string) {
	tc.Resources.mu.Lock()
	defer tc.Resources.mu.Unlock()
	tc.Resources.Subscriptions = append(tc.Resources.Subscriptions, subName)
}

// TrackTempFile records a temporary file created during the test
func (tc *TestContext) TrackTempFile(fileName string) {
	tc.Resources.mu.Lock()
	defer tc.Resources.mu.Unlock()
	tc.Resources.TempFiles = append(tc.Resources.TempFiles, fileName)
}

// TrackConnection records an open connection
func (tc *TestContext) TrackConnection(connInfo string) {
	tc.Resources.mu.Lock()
	defer tc.Resources.mu.Unlock()
	tc.Resources.OpenConnections = append(tc.Resources.OpenConnections, connInfo)
}

// UntrackConnection removes a connection from tracking (when closed)
func (tc *TestContext) UntrackConnection(connInfo string) {
	tc.Resources.mu.Lock()
	defer tc.Resources.mu.Unlock()

	// Remove the connection from the list
	for i, conn := range tc.Resources.OpenConnections {
		if conn == connInfo {
			tc.Resources.OpenConnections = append(
				tc.Resources.OpenConnections[:i],
				tc.Resources.OpenConnections[i+1:]...)
			break
		}
	}
}

// TrackCustomResource tracks a custom resource type
func (tc *TestContext) TrackCustomResource(resourceType, resourceID string) {
	tc.Resources.mu.Lock()
	defer tc.Resources.mu.Unlock()
	tc.Resources.CustomResources[resourceType] = append(
		tc.Resources.CustomResources[resourceType], resourceID)
}

// GetTrackedResources returns a snapshot of all tracked resources
func (tc *TestContext) GetTrackedResources() TestResourceSnapshot {
	tc.Resources.mu.Lock()
	defer tc.Resources.mu.Unlock()

	// Create copies to avoid race conditions
	snapshot := TestResourceSnapshot{
		Topics:          make([]string, len(tc.Resources.Topics)),
		Subscriptions:   make([]string, len(tc.Resources.Subscriptions)),
		TempFiles:       make([]string, len(tc.Resources.TempFiles)),
		OpenConnections: make([]string, len(tc.Resources.OpenConnections)),
		CustomResources: make(map[string][]string),
	}

	copy(snapshot.Topics, tc.Resources.Topics)
	copy(snapshot.Subscriptions, tc.Resources.Subscriptions)
	copy(snapshot.TempFiles, tc.Resources.TempFiles)
	copy(snapshot.OpenConnections, tc.Resources.OpenConnections)

	for k, v := range tc.Resources.CustomResources {
		snapshot.CustomResources[k] = make([]string, len(v))
		copy(snapshot.CustomResources[k], v)
	}

	return snapshot
}

// TestResourceSnapshot represents a point-in-time snapshot of tracked resources
type TestResourceSnapshot struct {
	Topics          []string
	Subscriptions   []string
	TempFiles       []string
	OpenConnections []string
	CustomResources map[string][]string
}

// HasLeaks checks if there are any resources that haven't been cleaned up
func (s TestResourceSnapshot) HasLeaks() bool {
	return len(s.Topics) > 0 ||
		len(s.Subscriptions) > 0 ||
		len(s.TempFiles) > 0 ||
		len(s.OpenConnections) > 0 ||
		len(s.CustomResources) > 0
}

// GetLeakSummary returns a human-readable summary of resource leaks
func (s TestResourceSnapshot) GetLeakSummary() string {
	if !s.HasLeaks() {
		return "No resource leaks detected"
	}

	summary := "Resource leaks detected:\n"
	if len(s.Topics) > 0 {
		summary += fmt.Sprintf("  - Topics: %d (%v)\n", len(s.Topics), s.Topics)
	}
	if len(s.Subscriptions) > 0 {
		summary += fmt.Sprintf("  - Subscriptions: %d (%v)\n", len(s.Subscriptions), s.Subscriptions)
	}
	if len(s.TempFiles) > 0 {
		summary += fmt.Sprintf("  - Temp Files: %d (%v)\n", len(s.TempFiles), s.TempFiles)
	}
	if len(s.OpenConnections) > 0 {
		summary += fmt.Sprintf("  - Open Connections: %d (%v)\n", len(s.OpenConnections), s.OpenConnections)
	}
	for resourceType, resources := range s.CustomResources {
		if len(resources) > 0 {
			summary += fmt.Sprintf("  - %s: %d (%v)\n", resourceType, len(resources), resources)
		}
	}

	return summary
}

// UntrackTopic removes a topic from tracking (after successful cleanup)
func (tc *TestContext) UntrackTopic(topicName string) {
	tc.Resources.mu.Lock()
	defer tc.Resources.mu.Unlock()

	for i, topic := range tc.Resources.Topics {
		if topic == topicName {
			tc.Resources.Topics = append(
				tc.Resources.Topics[:i],
				tc.Resources.Topics[i+1:]...)
			break
		}
	}
}

// UntrackSubscription removes a subscription from tracking (after successful cleanup)
func (tc *TestContext) UntrackSubscription(subName string) {
	tc.Resources.mu.Lock()
	defer tc.Resources.mu.Unlock()

	for i, sub := range tc.Resources.Subscriptions {
		if sub == subName {
			tc.Resources.Subscriptions = append(
				tc.Resources.Subscriptions[:i],
				tc.Resources.Subscriptions[i+1:]...)
			break
		}
	}
}

// UntrackTempFile removes a temp file from tracking (after successful cleanup)
func (tc *TestContext) UntrackTempFile(fileName string) {
	tc.Resources.mu.Lock()
	defer tc.Resources.mu.Unlock()

	for i, file := range tc.Resources.TempFiles {
		if file == fileName {
			tc.Resources.TempFiles = append(
				tc.Resources.TempFiles[:i],
				tc.Resources.TempFiles[i+1:]...)
			break
		}
	}
}

// UntrackCustomResource removes a custom resource from tracking (after successful cleanup)
func (tc *TestContext) UntrackCustomResource(resourceType, resourceID string) {
	tc.Resources.mu.Lock()
	defer tc.Resources.mu.Unlock()

	resources, exists := tc.Resources.CustomResources[resourceType]
	if !exists {
		return
	}

	for i, res := range resources {
		if res == resourceID {
			tc.Resources.CustomResources[resourceType] = append(
				resources[:i],
				resources[i+1:]...)
			break
		}
	}

	// Remove the resource type key if no resources left
	if len(tc.Resources.CustomResources[resourceType]) == 0 {
		delete(tc.Resources.CustomResources, resourceType)
	}
}

// ClearCustomResourceType removes all resources of a specific type from tracking
func (tc *TestContext) ClearCustomResourceType(resourceType string) {
	tc.Resources.mu.Lock()
	defer tc.Resources.mu.Unlock()

	delete(tc.Resources.CustomResources, resourceType)
}
