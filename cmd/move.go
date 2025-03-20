/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// moveCmd represents the move command
var moveCmd = &cobra.Command{
	Use:   "move",
	Short: "Moves messages from a source to a destination",
	Long: `Moves messages from a source to a destination.
Each message is polled, published, and acknowledged sequentially.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Parse flags
		sourceType, _ := cmd.Flags().GetString("source-type")
		destType, _ := cmd.Flags().GetString("destination-type")
		source, _ := cmd.Flags().GetString("source")
		destination, _ := cmd.Flags().GetString("destination")
		count, _ := cmd.Flags().GetInt("count")

		// Validate supported types
		if sourceType != "GCP_PUBSUB_SUBSCRIPTION" {
			fmt.Printf("Error: unsupported source type: %s. Supported: GCP_PUBSUB_SUBSCRIPTION\n", sourceType)
			return
		}
		if destType != "GCP_PUBSUB_TOPIC" {
			fmt.Printf("Error: unsupported destination type: %s. Supported: GCP_PUBSUB_TOPIC\n", destType)
			return
		}

		// Informational output
		fmt.Printf("Moving messages from %s (%s) to %s (%s)\n", source, sourceType, destination, destType)

		// If count is 0, simulate moving a default of 3 messages until exhausted
		total := count
		if total == 0 {
			total = 3
		}

		for i := 1; i <= total; i++ {
			fmt.Printf("Processing message %d:\n", i)
			// Simulate polling
			fmt.Printf(" - Polling message from %s\n", source)
			// Simulate publishing
			fmt.Printf(" - Publishing message to %s\n", destination)
			// Simulate acknowledge
			fmt.Printf(" - Acknowledging message at %s\n", source)
		}

		fmt.Println("Move operation completed.")
	},
}

func init() {
	rootCmd.AddCommand(moveCmd)

	// Define command flags
	moveCmd.Flags().String("source-type", "", "Message source type")
	moveCmd.Flags().String("destination-type", "", "Message destination type")
	moveCmd.Flags().String("source", "", "Source identifier (e.g. subscription)")
	moveCmd.Flags().String("destination", "", "Destination identifier (e.g. topic)")
	moveCmd.Flags().Int("count", 0, "Number of messages to move (0 for all)")

	// Make flags required except for count
	moveCmd.MarkFlagRequired("source-type")
	moveCmd.MarkFlagRequired("destination-type")
	moveCmd.MarkFlagRequired("source")
	moveCmd.MarkFlagRequired("destination")
}
