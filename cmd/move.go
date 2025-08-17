/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"os"

	"cloud.google.com/go/pubsub/v2"
	pubsubpb "cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"github.com/spf13/cobra"
)

// moveCmd represents the move command
var moveCmd = &cobra.Command{
	Use:   "move",
	Short: "Moves messages from a source to a destination",
	Long: `Moves messages from a source to a destination.
Each message is polled, published, and acknowledged sequentially.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.SetOutput(os.Stdout)
		// Parse flags
		sourceType, _ := cmd.Flags().GetString("source-type")
		destType, _ := cmd.Flags().GetString("destination-type")
		source, _ := cmd.Flags().GetString("source")
		destination, _ := cmd.Flags().GetString("destination")
		count, _ := cmd.Flags().GetInt("count")
		pollTimeoutSec, _ := cmd.Flags().GetInt("polling-timeout-seconds")

		// Validate supported types
		if sourceType != "GCP_PUBSUB_SUBSCRIPTION" {
			log.Printf("Error: unsupported source type: %s. Supported: GCP_PUBSUB_SUBSCRIPTION", sourceType)
			return
		}
		if destType != "GCP_PUBSUB_TOPIC" {
			log.Printf("Error: unsupported destination type: %s. Supported: GCP_PUBSUB_TOPIC", destType)
			return
		}

		// Informational output
		log.Printf("Moving messages from %s to %s", source, destination)

		ctx := context.Background()
		// Each message poll will use a 5-second timeout.

		// Extract subscription project from full resource name
		subParts := strings.Split(source, "/")
		if len(subParts) < 4 {
			log.Fatalf("Invalid subscription resource format: %s", source)
		}
		subProj := subParts[1]

		// Extract topic project from full resource name
		topicParts := strings.Split(destination, "/")
		if len(topicParts) < 4 {
			log.Fatalf("Invalid topic resource format: %s", destination)
		}
		topicProj := topicParts[1]

		// Create clients for both subscription and topic operations
		// Use the subscription project for the subscription client
		subClient, err := pubsub.NewClient(ctx, subProj)
		if err != nil {
			log.Fatalf("Failed to create subscription client: %v", err)
		}
		defer subClient.Close()

		// Create topic client (may be different project)
		var topicClient *pubsub.Client
		if topicProj == subProj {
			topicClient = subClient // Reuse same client if same project
		} else {
			topicClient, err = pubsub.NewClient(ctx, topicProj)
			if err != nil {
				log.Fatalf("Failed to create topic client: %v", err)
			}
			defer topicClient.Close()
		}

		// Create publisher for the destination topic
		publisher := topicClient.Publisher(destination)
		defer publisher.Stop()

		// Use the SubscriptionAdminClient for manual pull operations
		subscriberClient := subClient.SubscriptionAdminClient
		defer subscriberClient.Close()

		var mu sync.Mutex
		processed := 0

		// Loop to pull a single message with 5-second timeout per poll.
		for {
			pollCtx, pollCancel := context.WithTimeout(ctx, time.Duration(pollTimeoutSec)*time.Second)
			req := &pubsubpb.PullRequest{
				Subscription: source,
				MaxMessages:  1,
			}
			resp, err := subscriberClient.Pull(pollCtx, req)
			pollCancel()

			// Handle the response
			if err != nil {
				// Check for context deadline exceeded errors
				if err == context.DeadlineExceeded ||
					pollCtx.Err() == context.DeadlineExceeded ||
					strings.Contains(err.Error(), "DeadlineExceeded") ||
					strings.Contains(err.Error(), "context deadline exceeded") {
					break
				}
				log.Printf("Error during message pull: %v", err)
				continue
			}

			// Check if we got an empty response
			if resp == nil || len(resp.ReceivedMessages) == 0 {
				break
			}

			receivedMsg := resp.ReceivedMessages[0]
			// Atomically increment the processed count and assign message number.
			mu.Lock()
			processed++
			msgNum := processed
			mu.Unlock()

			log.Printf("Pulled message %d", msgNum)
			log.Printf("Publishing message %d", msgNum)
			result := publisher.Publish(ctx, &pubsub.Message{
				Data:       receivedMsg.Message.Data,
				Attributes: receivedMsg.Message.Attributes,
			})
			_, err = result.Get(ctx)
			if err != nil {
				log.Printf("Failed to publish message %d: %v", msgNum, err)
				continue
			}
			log.Printf("Published message %d successfully", msgNum)

			// Acknowledge the message.
			ackReq := &pubsubpb.AcknowledgeRequest{
				Subscription: source,
				AckIds:       []string{receivedMsg.AckId},
			}
			if err := subscriberClient.Acknowledge(ctx, ackReq); err != nil {
				log.Printf("Failed to ack message %d: %v", msgNum, err)
				continue
			}
			log.Printf("Acked message %d", msgNum)
			log.Printf("Processed message %d", msgNum)

			if count > 0 && processed >= count {
				break
			}
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

	// Define command flags
	moveCmd.Flags().String("source-type", "", "Message source type")
	moveCmd.Flags().String("destination-type", "", "Message destination type")
	moveCmd.Flags().String("source", "", "Full source resource name (e.g. projects/<proj>/subscriptions/<sub>)")
	moveCmd.Flags().String("destination", "", "Full destination resource name (e.g. projects/<proj>/topics/<topic>)")
	moveCmd.Flags().Int("count", 0, "Number of messages to move (0 for unlimited, continues until source is exhausted)")
	moveCmd.Flags().Int("polling-timeout-seconds", 10, "Timeout in seconds for polling a single message")

	// Make flags required except for count
	_ = moveCmd.MarkFlagRequired("source-type")
	_ = moveCmd.MarkFlagRequired("destination-type")
	_ = moveCmd.MarkFlagRequired("source")
	_ = moveCmd.MarkFlagRequired("destination")
}
