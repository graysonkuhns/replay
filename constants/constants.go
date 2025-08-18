package constants

import "time"

// Broker type constants
const (
	BrokerTypeGCPPubSubSubscription = "GCP_PUBSUB_SUBSCRIPTION"
	BrokerTypeGCPPubSubTopic        = "GCP_PUBSUB_TOPIC"
)

// Default configuration values
const (
	DefaultPollTimeoutSeconds = 10
	DefaultPollTimeout        = DefaultPollTimeoutSeconds * time.Second
	DefaultBatchSize          = 1
)

// SupportedSourceTypes defines the supported source broker types
var SupportedSourceTypes = []string{
	BrokerTypeGCPPubSubSubscription,
}

// SupportedDestinationTypes defines the supported destination broker types
var SupportedDestinationTypes = []string{
	BrokerTypeGCPPubSubTopic,
}
