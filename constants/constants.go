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
	TestShortPollTimeout    = 3 * time.Second
	TestRetryDelay          = 5 * time.Second
	TestMessagePropagation  = 15 * time.Second
	TestLongPollTimeout     = 20 * time.Second
	TestAckDeadlineExpiry   = 35 * time.Second
	TestExtendedPollTimeout = 30 * time.Second
)

// Test configuration
const (
	TestMaxRetries               = 3
	TestMessageRetentionDuration = 604800 * time.Second // 7 days
	TestMaxOutstandingOffset     = 10                   // Added to expected count for MaxOutstandingMessages
)

// Wait times for test operations
const (
	TestWaitShort  = 3 * time.Second
	TestWaitMedium = 7 * time.Second
	TestWaitLong   = 15 * time.Second
)
