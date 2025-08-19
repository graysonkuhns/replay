/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"os"

	"replay/logger"

	"github.com/spf13/cobra"
)

// MoveHandler implements MessageHandler for automatic message moving
type MoveHandler struct {
	broker MessageBroker
	log    logger.Logger
}

// NewMoveHandler creates a new move handler
func NewMoveHandler(broker MessageBroker) *MoveHandler {
	return &MoveHandler{
		broker: broker,
		log:    logger.NewLogger(),
	}
}

// HandleMessage implements automatic message moving
func (h *MoveHandler) HandleMessage(ctx context.Context, message *Message, msgNum int) (bool, error) {
	h.log.Info("Pulled message", logger.Int("messageNum", msgNum))
	h.log.Info("Publishing message", logger.Int("messageNum", msgNum))

	// Publish the message
	if err := h.broker.Publish(ctx, message); err != nil {
		h.log.Error("Failed to publish message", err, logger.Int("messageNum", msgNum))
		return false, fmt.Errorf("failed to publish: %w", err)
	}
	h.log.Info("Published message successfully", logger.Int("messageNum", msgNum))

	// Log acknowledgement (actual ack handled by processor)
	h.log.Info("Acked message", logger.Int("messageNum", msgNum))
	h.log.Info("Processed message", logger.Int("messageNum", msgNum))

	return true, nil
}

// moveCmd represents the move command
var moveCmd = &cobra.Command{
	Use:   "move",
	Short: "Moves messages from a source to a destination",
	Long: `Moves messages from a source to a destination.
Each message is polled, published, and acknowledged sequentially.`,
	Run: func(cmd *cobra.Command, args []string) {
		log := logger.NewLogger()

		// Parse and validate configuration
		config, err := ParseCommandConfig(cmd)
		if err != nil {
			log.Error("Configuration error", err)
			return
		}

		// Informational output
		log.Info("Moving messages",
			logger.String("source", config.Source),
			logger.String("destination", config.Destination))

		ctx := context.Background()

		// Create message broker
		broker, err := NewPubSubBroker(ctx, config.Source, config.Destination)
		if err != nil {
			log.Error("Failed to create broker", err)
			os.Exit(1)
		}
		defer broker.Close()

		// Create handler and processor
		handler := NewMoveHandler(broker)
		processor := NewMessageProcessor(broker, *config, handler, os.Stdout)

		// Process messages
		processed, err := processor.Process(ctx)
		if err != nil {
			log.Error("Error during processing", err)
		}

		log.Info("Move operation completed", logger.Int("totalMoved", processed))
	},
}

func init() {
	rootCmd.AddCommand(moveCmd)

	// Add common flags
	AddCommonFlags(moveCmd)

	// Override the count flag description for move command
	moveCmd.Flags().Lookup("count").Usage = "Number of messages to move (0 for unlimited, continues until source is exhausted)"
}
