package testhelpers

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// PollingOptions configures the behavior of polling functions
type PollingOptions struct {
	// InitialInterval is the initial delay between retries
	InitialInterval time.Duration
	// MaxInterval is the maximum delay between retries
	MaxInterval time.Duration
	// Multiplier is the factor by which the interval increases
	Multiplier float64
	// MaxElapsedTime is the maximum time to keep trying
	MaxElapsedTime time.Duration
	// ProgressCallback is called periodically to report progress
	ProgressCallback func(elapsed time.Duration, attempt int)
}

// DefaultPollingOptions returns sensible defaults for polling
func DefaultPollingOptions() *PollingOptions {
	return &PollingOptions{
		InitialInterval:  1 * time.Second,
		MaxInterval:      30 * time.Second,
		Multiplier:       1.5,
		MaxElapsedTime:   2 * time.Minute,
		ProgressCallback: nil,
	}
}

// WaitForCondition polls until the condition returns true or the timeout is reached
func WaitForCondition(ctx context.Context, description string, condition func() (bool, error), opts *PollingOptions) error {
	if opts == nil {
		opts = DefaultPollingOptions()
	}

	startTime := time.Now()
	attempt := 0
	currentInterval := opts.InitialInterval

	// Create a timeout context if none provided
	timeoutCtx, cancel := context.WithTimeout(ctx, opts.MaxElapsedTime)
	defer cancel()

	// Calculate progress interval based on max elapsed time
	progressInterval := opts.MaxElapsedTime / 10
	if progressInterval < 10*time.Millisecond {
		progressInterval = 10 * time.Millisecond
	}
	if progressInterval > 10*time.Second {
		progressInterval = 10 * time.Second
	}
	progressTicker := time.NewTicker(progressInterval)
	defer progressTicker.Stop()

	for {
		attempt++

		// Check condition
		ok, err := condition()
		if err != nil {
			return fmt.Errorf("condition check failed: %w", err)
		}
		if ok {
			return nil
		}

		// Check for timeout
		select {
		case <-timeoutCtx.Done():
			elapsed := time.Since(startTime)
			return fmt.Errorf("timed out after %v waiting for %s (attempts: %d)", elapsed, description, attempt)
		case <-progressTicker.C:
			// Report progress periodically
			if opts.ProgressCallback != nil {
				elapsed := time.Since(startTime)
				opts.ProgressCallback(elapsed, attempt)
			}
		default:
			// Continue polling
		}

		// Calculate next interval with exponential backoff
		if currentInterval < opts.MaxInterval {
			currentInterval = time.Duration(float64(currentInterval) * opts.Multiplier)
			if currentInterval > opts.MaxInterval {
				currentInterval = opts.MaxInterval
			}
		}

		// Wait before next attempt
		timer := time.NewTimer(currentInterval)
		select {
		case <-timeoutCtx.Done():
			timer.Stop()
			elapsed := time.Since(startTime)
			return fmt.Errorf("context cancelled after %v waiting for %s (attempts: %d)", elapsed, description, attempt)
		case <-timer.C:
			// Continue to next iteration
		}
	}
}

// WaitForMessagesInSubscription waits until the specified number of messages are available in a subscription
func WaitForMessagesInSubscription(ctx context.Context, base *BaseE2ETest, subscription string, expectedCount int, opts *PollingOptions) error {
	if opts == nil {
		opts = DefaultPollingOptions()
		// Customize for message waiting
		opts.MaxElapsedTime = 90 * time.Second
		opts.ProgressCallback = func(elapsed time.Duration, attempt int) {
			base.Logf("Waiting for %d messages in %s (elapsed: %v, attempts: %d)",
				expectedCount, subscription, elapsed, attempt)
		}
	}

	description := fmt.Sprintf("%d messages in subscription %s", expectedCount, subscription)

	condition := func() (bool, error) {
		// Use PollMessages to check availability
		messages, err := PollMessages(
			base.Setup.Context,
			base.Setup.Client,
			subscription,
			base.TestRunID,
			expectedCount,
		)

		// PollMessages already filters by testRunID and returns error if count doesn't match
		if err == nil && len(messages) == expectedCount {
			base.Logf("Found %d messages (expected %d) in %s", len(messages), expectedCount, subscription)
			return true, nil
		}

		if err != nil {
			// If it's just a count mismatch, log and continue
			if fmt.Sprintf("%v", err) == fmt.Sprintf("expected %d messages, got %d", expectedCount, len(messages)) {
				base.Logf("Found only %d messages (expected %d) in %s, continuing to wait...",
					len(messages), expectedCount, subscription)
				return false, nil
			}
			// For other errors, return them
			return false, fmt.Errorf("failed to poll messages: %w", err)
		}

		return false, nil
	}

	return WaitForCondition(ctx, description, condition, opts)
}

// WaitWithBackoff performs a simple wait with exponential backoff
// This is useful for replacing time.Sleep() in tests where we're not checking a condition
func WaitWithBackoff(ctx context.Context, description string, duration time.Duration, base *BaseE2ETest) error {
	// Create a new context with the specified duration as timeout
	waitCtx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	opts := &PollingOptions{
		InitialInterval: duration / 10, // Start with 1/10th of total duration
		MaxInterval:     duration / 2,  // Max interval is half the duration
		Multiplier:      2.0,
		MaxElapsedTime:  duration,
		ProgressCallback: func(elapsed time.Duration, attempt int) {
			remaining := duration - elapsed
			if remaining > 0 {
				base.Logf("Waiting for %s (remaining: %v)", description, remaining)
			}
		},
	}

	// Simple condition that always returns false until timeout
	condition := func() (bool, error) {
		return false, nil
	}

	err := WaitForCondition(waitCtx, description, condition, opts)

	// Check if the parent context was cancelled (not our timeout)
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// If we get a timeout error from our own timeout, that's expected for a simple wait
	if err != nil && strings.Contains(err.Error(), "timed out after") && strings.Contains(err.Error(), description) {
		return nil
	}

	// For our own context timeout, also return nil
	if err != nil && waitCtx.Err() == context.DeadlineExceeded {
		return nil
	}

	return err
}
