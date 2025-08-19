package constants

import "time"

// Broker types
const (
	BrokerTypeGCPPubSubSubscription = "GCP_PUBSUB_SUBSCRIPTION"
	BrokerTypeGCPPubSubTopic        = "GCP_PUBSUB_TOPIC"
)

// Default configuration values
const (
	DefaultPollTimeoutSeconds = 10
	DefaultPollTimeout        = 10 * time.Second
	DefaultMaxMessages        = 1
)

// Test-specific timeouts
const (
	TestShortPollTimeout    = 5 * time.Second
	TestRetryDelay          = 10 * time.Second
	TestMessagePropagation  = 30 * time.Second
	TestLongPollTimeout     = 60 * time.Second
	TestAckDeadlineExpiry   = 70 * time.Second
	TestExtendedPollTimeout = 90 * time.Second
)

// Test configuration
const (
	TestMaxRetries               = 3
	TestMessageRetentionDuration = 604800 * time.Second // 7 days
	TestMaxOutstandingOffset     = 10                   // Added to expected count for MaxOutstandingMessages
)

// Wait times for test operations
const (
	TestWaitShort  = 10 * time.Second
	TestWaitMedium = 20 * time.Second
	TestWaitLong   = 30 * time.Second
)
