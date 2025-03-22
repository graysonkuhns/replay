/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/pubsub"
	pubsubapiv1 "cloud.google.com/go/pubsub/apiv1"
	"github.com/spf13/cobra"
	pubsubpb "google.golang.org/genproto/googleapis/pubsub/v1"
)

// dlrCmd represents the dlr command
var dlrCmd = &cobra.Command{
	Use:   "dlr",
	Short: "Review and process dead-lettered messages",
	Long: `Interactively review dead-lettered messages and choose to discard or move each message.
For moved messages, the message is republished to the destination.`,
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
		fmt.Printf("Starting DLR review from %s\n", source)
		ctx := context.Background()

		// Set up subscription client
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

		// Set up topic client
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

		// Create low-level Subscriber client
		subscriberClient, err := pubsubapiv1.NewSubscriberClient(ctx)
		if err != nil {
			log.Fatalf("Failed to create subscriber client: %v", err)
		}
		defer subscriberClient.Close()

		reader := bufio.NewReader(os.Stdin)
		processed := 0

		// Loop to pull messages interactively
		for {
			pollCtx, pollCancel := context.WithTimeout(ctx, 5*time.Second)
			req := &pubsubpb.PullRequest{
				Subscription: source,
				MaxMessages:  1,
			}
			resp, err := subscriberClient.Pull(pollCtx, req)
			pollCancel()
			if err != nil {
				if strings.Contains(err.Error(), "DeadlineExceeded") {
					log.Printf("No messages received within timeout")
					break
				}
				log.Printf("Error during message pull: %v", err)
				continue
			}
			if len(resp.ReceivedMessages) == 0 {
				log.Printf("No messages received")
				break
			}
			receivedMsg := resp.ReceivedMessages[0]
			processed++
			msgNum := processed

			// Show message details and prompt for action
			fmt.Printf("\nMessage %d:\n", msgNum)
			fmt.Printf("Data: %s\n", string(receivedMsg.Message.Data))
			fmt.Printf("Attributes: %v\n", receivedMsg.Message.Attributes)
			fmt.Print("Choose action ([m]ove / [d]iscard): ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(strings.ToLower(input))
			if input == "m" {
				fmt.Printf("Publishing message %d...\n", msgNum)
				result := topic.Publish(ctx, &pubsub.Message{
					Data:       receivedMsg.Message.Data,
					Attributes: receivedMsg.Message.Attributes,
				})
				_, err := result.Get(ctx)
				if err != nil {
					log.Printf("Failed to publish message %d: %v", msgNum, err)
					continue
				}
				fmt.Printf("Message %d moved successfully\n", msgNum)
			} else if input == "d" {
				fmt.Printf("Message %d discarded\n", msgNum)
			} else {
				fmt.Printf("Invalid input. Skipping message %d\n", msgNum)
			}
			// Acknowledge the message
			ackReq := &pubsubpb.AcknowledgeRequest{
				Subscription: source,
				AckIds:       []string{receivedMsg.AckId},
			}
			if err := subscriberClient.Acknowledge(ctx, ackReq); err != nil {
				log.Printf("Failed to acknowledge message %d: %v", msgNum, err)
			}
			if count > 0 && processed >= count {
				break
			}
		}

		fmt.Printf("\nDLR review completed. Total messages processed: %d\n", processed)
	},
}

func init() {
	rootCmd.AddCommand(dlrCmd)

	// Define flags similar to move command flags
	dlrCmd.Flags().String("source-type", "", "Message source type")
	dlrCmd.Flags().String("destination-type", "", "Message destination type")
	dlrCmd.Flags().String("source", "", "Full source resource name (e.g. projects/<proj>/subscriptions/<sub>)")
	dlrCmd.Flags().String("destination", "", "Full destination resource name (e.g. projects/<proj>/topics/<topic>)")
	dlrCmd.Flags().Int("count", 0, "Number of messages to process (0 for all messages)")
	dlrCmd.MarkFlagRequired("source-type")
	dlrCmd.MarkFlagRequired("destination-type")
	dlrCmd.MarkFlagRequired("source")
	dlrCmd.MarkFlagRequired("destination")
}
