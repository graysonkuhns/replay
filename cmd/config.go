package cmd

import (
	"fmt"
	"time"

	"replay/constants"

	"github.com/spf13/cobra"
)

// CommandConfig holds the configuration for message processing commands
type CommandConfig struct {
	SourceType      string
	DestinationType string
	Source          string
	Destination     string
	Count           int
	PollTimeout     time.Duration
	PrettyJSON      bool
}

// ParseCommandConfig extracts and validates command configuration from cobra command
func ParseCommandConfig(cmd *cobra.Command) (*CommandConfig, error) {
	sourceType, _ := cmd.Flags().GetString("source-type")
	destType, _ := cmd.Flags().GetString("destination-type")
	source, _ := cmd.Flags().GetString("source")
	destination, _ := cmd.Flags().GetString("destination")
	count, _ := cmd.Flags().GetInt("count")
	pollTimeoutSec, _ := cmd.Flags().GetInt("polling-timeout-seconds")

	// Check if pretty-json flag exists (for dlr command)
	prettyJSON := false
	if cmd.Flags().Lookup("pretty-json") != nil {
		prettyJSON, _ = cmd.Flags().GetBool("pretty-json")
	}

	// Validate supported types
	if sourceType != constants.BrokerTypeGCPPubSubSubscription {
		return nil, fmt.Errorf("unsupported source type: %s. Supported: %s", sourceType, constants.BrokerTypeGCPPubSubSubscription)
	}
	if destType != constants.BrokerTypeGCPPubSubTopic {
		return nil, fmt.Errorf("unsupported destination type: %s. Supported: %s", destType, constants.BrokerTypeGCPPubSubTopic)
	}

	return &CommandConfig{
		SourceType:      sourceType,
		DestinationType: destType,
		Source:          source,
		Destination:     destination,
		Count:           count,
		PollTimeout:     time.Duration(pollTimeoutSec) * time.Second,
		PrettyJSON:      prettyJSON,
	}, nil
}

// AddCommonFlags adds common flags to a cobra command
func AddCommonFlags(cmd *cobra.Command) {
	cmd.Flags().String("source-type", "", "Message source type")
	cmd.Flags().String("destination-type", "", "Message destination type")
	cmd.Flags().String("source", "", "Full source resource name (e.g. projects/<proj>/subscriptions/<sub>)")
	cmd.Flags().String("destination", "", "Full destination resource name (e.g. projects/<proj>/topics/<topic>)")
	cmd.Flags().Int("count", 0, "Number of messages to process (0 for all messages)")
	cmd.Flags().Int("polling-timeout-seconds", constants.DefaultPollTimeoutSeconds, "Timeout in seconds for polling a single message")

	_ = cmd.MarkFlagRequired("source-type")
	_ = cmd.MarkFlagRequired("destination-type")
	_ = cmd.MarkFlagRequired("source")
	_ = cmd.MarkFlagRequired("destination")
}
