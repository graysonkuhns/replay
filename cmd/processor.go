package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"replay/constants"
	"replay/logger"
)

// ErrQuit is returned when the user chooses to quit
var ErrQuit = errors.New("user quit")

// MessageHandler defines how to handle each message
type MessageHandler interface {
	// HandleMessage processes a message and returns whether to acknowledge it
	HandleMessage(ctx context.Context, message *Message, msgNum int) (acknowledge bool, err error)
}

// MessageProcessor handles the common logic for processing messages
type MessageProcessor struct {
	broker  MessageBroker
	config  CommandConfig
	handler MessageHandler
	output  io.Writer
	log     logger.Logger
}

// NewMessageProcessor creates a new message processor
func NewMessageProcessor(broker MessageBroker, config CommandConfig, handler MessageHandler, output io.Writer) *MessageProcessor {
	return &MessageProcessor{
		broker:  broker,
		config:  config,
		handler: handler,
		output:  output,
		log:     logger.NewLoggerWithOutput(output),
	}
}

// Process runs the message processing loop
func (p *MessageProcessor) Process(ctx context.Context) (int, error) {
	processed := 0

	for {
		// Pull a message
		message, err := p.broker.Pull(ctx, PullConfig{
			MaxMessages: constants.DefaultMaxMessages,
			Timeout:     p.config.PollTimeout,
		})

		// Handle pull errors
		if err != nil {
			if strings.Contains(err.Error(), "DeadlineExceeded") ||
				errors.Is(err, context.DeadlineExceeded) {
				break
			}
			p.log.Error("Error during message pull", err)
			continue
		}

		// No more messages
		if message == nil {
			break
		}

		msgNum := processed + 1

		// Handle the message
		acknowledge, err := p.handler.HandleMessage(ctx, message, msgNum)
		if err != nil {
			// Check if it's a quit error
			if errors.Is(err, ErrQuit) {
				break
			}
			p.log.Error("Error handling message", err, logger.Int("messageNum", msgNum))
			continue
		}

		// Acknowledge if requested
		if acknowledge {
			if err := p.broker.Acknowledge(ctx, message.AckID); err != nil {
				p.log.Error("Warning: failed to acknowledge message", err, logger.Int("messageNum", msgNum))
			}
			processed++
		}

		// Check if we've reached the count limit
		if p.config.Count > 0 && processed >= p.config.Count {
			break
		}
	}

	return processed, nil
}

// FormatMessageData formats message data for display
func FormatMessageData(data []byte, prettyJSON bool) string {
	if !prettyJSON {
		return string(data)
	}

	var jsonData interface{}
	if err := json.Unmarshal(data, &jsonData); err == nil {
		if prettyBytes, err := json.MarshalIndent(jsonData, "", "  "); err == nil {
			return string(prettyBytes)
		}
	}
	return string(data)
}
