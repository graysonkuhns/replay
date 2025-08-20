package testhelpers

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWaitForCondition(t *testing.T) {
	t.Run("returns immediately when condition is true", func(t *testing.T) {
		called := 0
		condition := func() (bool, error) {
			called++
			return true, nil
		}

		err := WaitForCondition(context.Background(), "test condition", condition, nil)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if called != 1 {
			t.Errorf("Expected condition to be called once, was called %d times", called)
		}
	})

	t.Run("retries until condition becomes true", func(t *testing.T) {
		called := 0
		condition := func() (bool, error) {
			called++
			return called >= 3, nil
		}

		opts := &PollingOptions{
			InitialInterval: 10 * time.Millisecond,
			MaxInterval:     50 * time.Millisecond,
			Multiplier:      2.0,
			MaxElapsedTime:  1 * time.Second,
		}

		err := WaitForCondition(context.Background(), "test condition", condition, opts)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if called != 3 {
			t.Errorf("Expected condition to be called 3 times, was called %d times", called)
		}
	})

	t.Run("returns error when condition returns error", func(t *testing.T) {
		expectedErr := errors.New("condition failed")
		condition := func() (bool, error) {
			return false, expectedErr
		}

		err := WaitForCondition(context.Background(), "test condition", condition, nil)
		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !errors.Is(err, expectedErr) {
			t.Errorf("Expected error to wrap %v, got: %v", expectedErr, err)
		}
	})

	t.Run("times out when condition never becomes true", func(t *testing.T) {
		called := 0
		condition := func() (bool, error) {
			called++
			return false, nil
		}

		opts := &PollingOptions{
			InitialInterval: 10 * time.Millisecond,
			MaxInterval:     20 * time.Millisecond,
			Multiplier:      1.5,
			MaxElapsedTime:  100 * time.Millisecond,
		}

		start := time.Now()
		err := WaitForCondition(context.Background(), "test condition", condition, opts)
		elapsed := time.Since(start)

		if err == nil {
			t.Error("Expected timeout error, got nil")
		}
		if elapsed < opts.MaxElapsedTime {
			t.Errorf("Expected to wait at least %v, but only waited %v", opts.MaxElapsedTime, elapsed)
		}
		if called < 2 {
			t.Errorf("Expected condition to be called multiple times, was called %d times", called)
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		called := 0
		condition := func() (bool, error) {
			called++
			if called == 2 {
				cancel() // Cancel after second call
			}
			return false, nil
		}

		opts := &PollingOptions{
			InitialInterval: 50 * time.Millisecond,
			MaxInterval:     100 * time.Millisecond,
			Multiplier:      2.0,
			MaxElapsedTime:  5 * time.Second,
		}

		err := WaitForCondition(ctx, "test condition", condition, opts)
		if err == nil {
			t.Error("Expected context cancellation error, got nil")
		}
		if called < 2 {
			t.Errorf("Expected condition to be called at least twice before cancellation, was called %d times", called)
		}
	})

	t.Run("calls progress callback", func(t *testing.T) {
		progressCalled := 0
		var lastElapsed time.Duration
		var lastAttempt int

		opts := &PollingOptions{
			InitialInterval: 10 * time.Millisecond,
			MaxInterval:     20 * time.Millisecond,
			Multiplier:      1.5,
			MaxElapsedTime:  200 * time.Millisecond,
			ProgressCallback: func(elapsed time.Duration, attempt int) {
				progressCalled++
				lastElapsed = elapsed
				lastAttempt = attempt
			},
		}

		condition := func() (bool, error) {
			return false, nil // Never succeeds
		}

		_ = WaitForCondition(context.Background(), "test condition", condition, opts)

		if progressCalled == 0 {
			t.Error("Expected progress callback to be called at least once")
		}
		if lastElapsed == 0 {
			t.Error("Expected elapsed time to be non-zero in progress callback")
		}
		if lastAttempt == 0 {
			t.Error("Expected attempt count to be non-zero in progress callback")
		}
	})

	t.Run("exponential backoff works correctly", func(t *testing.T) {
		var intervals []time.Duration
		lastCall := time.Now()

		condition := func() (bool, error) {
			now := time.Now()
			if !lastCall.IsZero() {
				intervals = append(intervals, now.Sub(lastCall))
			}
			lastCall = now
			return false, nil
		}

		opts := &PollingOptions{
			InitialInterval: 10 * time.Millisecond,
			MaxInterval:     100 * time.Millisecond,
			Multiplier:      2.0,
			MaxElapsedTime:  300 * time.Millisecond,
		}

		_ = WaitForCondition(context.Background(), "test condition", condition, opts)

		if len(intervals) < 2 {
			t.Fatal("Expected at least 2 intervals to verify backoff")
		}

		// Verify intervals are increasing (with some tolerance for timing)
		for i := 1; i < len(intervals); i++ {
			if intervals[i] < intervals[i-1] {
				t.Errorf("Expected interval %d (%v) to be >= interval %d (%v)",
					i, intervals[i], i-1, intervals[i-1])
			}
		}

		// Verify we don't exceed max interval (with 20ms tolerance)
		for i, interval := range intervals {
			if interval > opts.MaxInterval+20*time.Millisecond {
				t.Errorf("Interval %d (%v) exceeded max interval (%v)",
					i, interval, opts.MaxInterval)
			}
		}
	})
}

func TestWaitWithBackoff(t *testing.T) {
	t.Run("waits for specified duration", func(t *testing.T) {
		base := &BaseE2ETest{T: t}
		duration := 100 * time.Millisecond

		start := time.Now()
		err := WaitWithBackoff(context.Background(), "test wait", duration, base)
		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Allow some tolerance for timing
		tolerance := 50 * time.Millisecond
		if elapsed < duration-tolerance || elapsed > duration+tolerance {
			t.Errorf("Expected to wait approximately %v, but waited %v", duration, elapsed)
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		base := &BaseE2ETest{T: t}
		ctx, cancel := context.WithCancel(context.Background())

		// Cancel after 50ms
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		start := time.Now()
		err := WaitWithBackoff(ctx, "test wait", 5*time.Second, base)
		elapsed := time.Since(start)

		if err == nil {
			t.Error("Expected context cancellation error, got nil")
		}

		// Should have been canceled well before the full duration
		if elapsed > 200*time.Millisecond {
			t.Errorf("Expected to be canceled quickly, but waited %v", elapsed)
		}
	})
}

// MockBaseE2ETest is a minimal implementation for testing
type MockBaseE2ETest struct {
	*testing.T
}

func (m *MockBaseE2ETest) Logf(format string, args ...interface{}) {
	m.T.Logf(format, args...)
}

func (m *MockBaseE2ETest) Helper() {
	m.T.Helper()
}

func TestDefaultPollingOptions(t *testing.T) {
	opts := DefaultPollingOptions()

	if opts.InitialInterval != 1*time.Second {
		t.Errorf("Expected initial interval of 1s, got %v", opts.InitialInterval)
	}
	if opts.MaxInterval != 30*time.Second {
		t.Errorf("Expected max interval of 30s, got %v", opts.MaxInterval)
	}
	if opts.Multiplier != 1.5 {
		t.Errorf("Expected multiplier of 1.5, got %v", opts.Multiplier)
	}
	if opts.MaxElapsedTime != 2*time.Minute {
		t.Errorf("Expected max elapsed time of 2m, got %v", opts.MaxElapsedTime)
	}
}
