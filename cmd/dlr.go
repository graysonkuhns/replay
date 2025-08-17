/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// DLRHandler implements MessageHandler for interactive dead-letter review
type DLRHandler struct {
	broker MessageBroker
	config CommandConfig
	reader *bufio.Reader
	output io.Writer
}

// NewDLRHandler creates a new DLR handler
func NewDLRHandler(broker MessageBroker, config CommandConfig) *DLRHandler {
	return &DLRHandler{
		broker: broker,
		config: config,
		reader: bufio.NewReader(os.Stdin),
		output: os.Stdout,
	}
}

// HandleMessage implements the interactive message handling for DLR
func (h *DLRHandler) HandleMessage(ctx context.Context, message *Message, msgNum int) (bool, error) {
	// Display message details
	fmt.Fprintf(h.output, "\nMessage %d:\n", msgNum)

	// Format and display message data
	dataStr := FormatMessageData(message.Data, h.config.PrettyJSON)
	if h.config.PrettyJSON && strings.HasPrefix(dataStr, "{") {
		fmt.Fprintf(h.output, "Data (pretty JSON):\n%s\n", dataStr)
	} else {
		fmt.Fprintf(h.output, "Data:\n%s\n", dataStr)
	}
	fmt.Fprintf(h.output, "Attributes: %v\n", message.Attributes)

	// Interactive prompt loop
	for {
		fmt.Fprint(h.output, "Choose action ([m]ove / [d]iscard / [q]uit): ")
		input, _ := h.reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "m":
			// Move the message
			if err := h.broker.Publish(ctx, message); err != nil {
				return false, fmt.Errorf("failed to move message %d", msgNum)
			}
			fmt.Fprintf(h.output, "Message %d moved successfully\n", msgNum)
			return true, nil

		case "d":
			// Discard the message
			fmt.Fprintf(h.output, "Message %d discarded (acked)\n", msgNum)
			return true, nil

		case "q":
			// Quit without acknowledging
			fmt.Fprintln(h.output, "Quitting review...")
			return false, ErrQuit

		default:
			fmt.Fprintln(h.output, "Invalid input. Please enter 'm', 'd', or 'q'.")
		}
	}
}

// dlrCmd represents the dlr command
var dlrCmd = &cobra.Command{
	Use:   "dlr",
	Short: "Review and process dead-lettered messages",
	Long: `Interactively review dead-lettered messages and choose to discard or move each message.
For moved messages, the message is republished to the destination.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Parse and validate configuration
		config, err := ParseCommandConfig(cmd)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		fmt.Printf("Starting DLR review from %s\n", config.Source)
		ctx := context.Background()

		// Create message broker
		broker, err := NewPubSubBroker(ctx, config.Source, config.Destination)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		defer broker.Close()

		// Create handler and processor
		handler := NewDLRHandler(broker, *config)
		processor := NewMessageProcessor(broker, *config, handler, os.Stdout)

		// Process messages
		processed, _ := processor.Process(ctx)

		fmt.Printf("\nDead-lettered messages review completed. Total messages processed: %d\n", processed)
	},
}

func init() {
	rootCmd.AddCommand(dlrCmd)

	// Add common flags
	AddCommonFlags(dlrCmd)

	// Add DLR-specific flags
	dlrCmd.Flags().Bool("pretty-json", false, "Display message data as pretty JSON")
}
