/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bufio"
	"context"
	"encoding/json" // added
	"fmt"
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
		pretty, _ := cmd.Flags().GetBool("pretty-json")
		pollTimeoutSec, _ := cmd.Flags().GetInt("polling-timeout-seconds")
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
			fmt.Printf("Error: Invalid subscription resource format: %s\n", source)
			return
		}
		subProj := subParts[1]
		subClient, err := pubsub.NewClient(ctx, subProj)
		if err != nil {
			fmt.Printf("Error: Failed to create subscription client: %v\n", err)
			return
		}
		defer subClient.Close()

		// Set up topic client
		topicParts := strings.Split(destination, "/")
		if len(topicParts) < 4 {
			fmt.Printf("Error: Invalid topic resource format: %s\n", destination)
			return
		}
		topicProj := topicParts[1]
		topicClient, err := pubsub.NewClient(ctx, topicProj)
		if err != nil {
			fmt.Printf("Error: Failed to create topic client: %v\n", err)
			return
		}
		defer topicClient.Close()
		topic := topicClient.Topic(topicParts[3])

		// Create low-level Subscriber client
		subscriberClient, err := pubsubapiv1.NewSubscriberClient(ctx)
		if err != nil {
			fmt.Printf("Error: Failed to create subscriber client: %v\n", err)
			return
		}
		defer subscriberClient.Close()

		reader := bufio.NewReader(os.Stdin)
		processed := 0

		// Loop to pull messages interactively
		for {
			msgNum := processed + 1
			pollCtx, pollCancel := context.WithTimeout(ctx, time.Duration(pollTimeoutSec)*time.Second)
			req := &pubsubpb.PullRequest{
				Subscription: source,
				MaxMessages:  1,
			}
			resp, err := subscriberClient.Pull(pollCtx, req)
			pollCancel()
			if err != nil {
				if strings.Contains(err.Error(), "DeadlineExceeded") {
					break
				}
				continue
			}
			if len(resp.ReceivedMessages) == 0 {
				break
			}
			receivedMsg := resp.ReceivedMessages[0]

			// Show message details and prompt for action
			fmt.Printf("\nMessage %d:\n", msgNum)
			if pretty {
				var jsonData interface{}
				if err := json.Unmarshal(receivedMsg.Message.Data, &jsonData); err == nil {
					if prettyBytes, err := json.MarshalIndent(jsonData, "", "  "); err == nil {
						fmt.Printf("Data (pretty JSON):\n%s\n", string(prettyBytes))
					} else {
						fmt.Printf("Data:\n%s\n", string(receivedMsg.Message.Data))
					}
				} else {
					fmt.Printf("Data:\n%s\n", string(receivedMsg.Message.Data))
				}
			} else {
				fmt.Printf("Data:\n%s\n", string(receivedMsg.Message.Data))
			}
			fmt.Printf("Attributes: %v\n", receivedMsg.Message.Attributes)

			// Keep asking for input until a valid option is selected
			var input string
			for {
				fmt.Print("Choose action ([m]ove / [d]iscard / [q]uit): ")
				input, _ = reader.ReadString('\n')
				input = strings.TrimSpace(strings.ToLower(input))
				if input == "m" || input == "d" || input == "q" {
					break
				}
				fmt.Printf("Invalid input. Please enter 'm', 'd', or 'q'.\n")
			}

			if input == "m" {
				result := topic.Publish(ctx, &pubsub.Message{
					Data:       receivedMsg.Message.Data,
					Attributes: receivedMsg.Message.Attributes,
				})
				_, err := result.Get(ctx)
				if err != nil {
					fmt.Printf("Failed to move message %d\n", msgNum)
					continue
				}
				fmt.Printf("Message %d moved successfully\n", msgNum)
			} else if input == "d" {
				fmt.Printf("Message %d discarded (acked)\n", msgNum)
			} else if input == "q" {
				fmt.Printf("Quitting review...\n")
				// When quitting, do not acknowledge the message
				// We need to explicitly log this so the test can verify the behavior
				break
			}

			// Only acknowledge the message if the user chose to move or discard it
			// Skip acknowledgement when quitting to keep the message in the subscription
			if input == "m" || input == "d" {
				// Acknowledge the message
				ackReq := &pubsubpb.AcknowledgeRequest{
					Subscription: source,
					AckIds:       []string{receivedMsg.AckId},
				}
				if err := subscriberClient.Acknowledge(ctx, ackReq); err != nil {
					// Failed to acknowledge, but continue processing
				}

				processed++
			}

			if count > 0 && processed >= count {
				break
			}
		}
		fmt.Printf("\nDead-lettered messages review completed. Total messages processed: %d\n", processed)
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
	dlrCmd.Flags().Bool("pretty-json", false, "Display message data as pretty JSON")
	dlrCmd.Flags().Int("polling-timeout-seconds", 5, "Timeout in seconds for polling a single message")
	dlrCmd.MarkFlagRequired("source-type")
	dlrCmd.MarkFlagRequired("destination-type")
	dlrCmd.MarkFlagRequired("source")
	dlrCmd.MarkFlagRequired("destination")
}
