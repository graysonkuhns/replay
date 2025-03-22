/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"cloud.google.com/go/pubsub"
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
		fmt.Printf("Moving messages from %s to %s\n", source, destination)

		// If count is 0, process a default of 3 messages
		total := count
		if total == 0 {
			total = 3
		}

		ctx := context.Background()
		// Extract subscription project from full resource name
		subParts := strings.Split(source, "/")
		if len(subParts) < 4 {
			log.Fatalf("Invalid subscription resource format: %s", source)
		}
		subProj := subParts[1]
		subClient, err := pubsub.NewClient(ctx, subProj)
		if err != nil {
			log.Fatalf("Failed to create subscription client: %v", err)
		}
		defer subClient.Close()
		sub := subClient.Subscription(subParts[3])

		// Extract topic project from full resource name
		topicParts := strings.Split(destination, "/")
		if len(topicParts) < 4 {
			log.Fatalf("Invalid topic resource format: %s", destination)
		}
		topicProj := topicParts[1]
		topicClient, err := pubsub.NewClient(ctx, topicProj)
		if err != nil {
			log.Fatalf("Failed to create topic client: %v", err)
		}
		defer topicClient.Close()
		topic := topicClient.Topic(topicParts[3])

		// Receive messages sequentially until total is reached
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		var mu sync.Mutex
		processed := 0

		err = sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
			result := topic.Publish(ctx, &pubsub.Message{
				Data:       msg.Data,
				Attributes: msg.Attributes,
			})
			_, err := result.Get(ctx)
			if err != nil {
				log.Printf("Failed to publish message: %v", err)
				msg.Nack()
				return
			}

			msg.Ack()
			fmt.Printf("Processed message %d\n", processed+1)

			mu.Lock()
			processed++
			if processed >= total {
				cancel()
			}
			mu.Unlock()
		})
		if err != nil {
			log.Printf("Error during message receive: %v", err)
		}

		fmt.Println("Move operation completed.")
	},
}

func init() {
	rootCmd.AddCommand(moveCmd)

	// Define command flags
	moveCmd.Flags().String("source-type", "", "Message source type")
	moveCmd.Flags().String("destination-type", "", "Message destination type")
	moveCmd.Flags().String("source", "", "Full source resource name (e.g. projects/<proj>/subscriptions/<sub>)")
	moveCmd.Flags().String("destination", "", "Full destination resource name (e.g. projects/<proj>/topics/<topic>)")
	moveCmd.Flags().Int("count", 0, "Number of messages to move (0 for default 3)")

	// Make flags required except for count
	moveCmd.MarkFlagRequired("source-type")
	moveCmd.MarkFlagRequired("destination-type")
	moveCmd.MarkFlagRequired("source")
	moveCmd.MarkFlagRequired("destination")
}
