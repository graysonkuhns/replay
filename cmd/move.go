/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

// MoveHandler implements MessageHandler for automatic message moving
type MoveHandler struct {
	broker MessageBroker
	logger *log.Logger
}

// NewMoveHandler creates a new move handler
func NewMoveHandler(broker MessageBroker) *MoveHandler {
	logger := log.New(os.Stdout, "", log.LstdFlags)
	return &MoveHandler{
		broker: broker,
		logger: logger,
	}
}

// HandleMessage implements automatic message moving
func (h *MoveHandler) HandleMessage(ctx context.Context, message *Message, msgNum int) (bool, error) {
	h.logger.Printf("Pulled message %d", msgNum)
	h.logger.Printf("Publishing message %d", msgNum)

	// Publish the message
	if err := h.broker.Publish(ctx, message); err != nil {
		h.logger.Printf("Failed to publish message %d: %v", msgNum, err)
		return false, fmt.Errorf("failed to publish: %w", err)
	}
	h.logger.Printf("Published message %d successfully", msgNum)

	// Log acknowledgement (actual ack handled by processor)
	h.logger.Printf("Acked message %d", msgNum)
	h.logger.Printf("Processed message %d", msgNum)

	return true, nil
}

// moveCmd represents the move command
var moveCmd = &cobra.Command{
	Use:   "move",
	Short: "Moves messages from a source to a destination",
	Long: `Moves messages from a source to a destination.
Each message is polled, published, and acknowledged sequentially.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.SetOutput(os.Stdout)

		// Parse and validate configuration
		config, err := ParseCommandConfig(cmd)
		if err != nil {
			log.Printf("Error: %v", err)
			return
		}

		// Informational output
		log.Printf("Moving messages from %s to %s", config.Source, config.Destination)

		ctx := context.Background()

		// Create message broker
		broker, err := NewPubSubBroker(ctx, config.Source, config.Destination)
		if err != nil {
			log.Fatalf("%v", err)
		}
		defer broker.Close()

		// Create handler and processor
		handler := NewMoveHandler(broker)
		processor := NewMessageProcessor(broker, *config, handler, os.Stdout)

		// Process messages
		processed, err := processor.Process(ctx)
		if err != nil {
			log.Printf("Error during processing: %v", err)
		}

		log.Printf("Move operation completed. Total messages moved: %d", processed)

		// Ensure all log output is flushed before exiting
		if f, ok := log.Writer().(*os.File); ok {
			_ = f.Sync()
		}
	},
}

func init() {
	rootCmd.AddCommand(moveCmd)

	// Add common flags
	AddCommonFlags(moveCmd)

	// Override the count flag description for move command
	moveCmd.Flags().Lookup("count").Usage = "Number of messages to move (0 for unlimited, continues until source is exhausted)"
}
