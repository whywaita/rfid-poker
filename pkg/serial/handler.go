package serial

import (
	"fmt"
	"log/slog"

	"github.com/goccy/go-json"
)

// MessageHandler is an interface for handling messages from M5Stack wired client
type MessageHandler interface {
	HandleBoot(portName string, msg BootMessage)
	HandleCard(portName string, msg CardMessage)
	HandleError(portName string, msg ErrorMessage)
}

// ParseAndHandle parses a JSON line and calls the appropriate handler
func ParseAndHandle(portName string, line string, handler MessageHandler) error {
	// validation check
	if _, err := json.Marshal(line); err != nil {
		return fmt.Errorf("failed to marshal line: %w", err)
	}

	// First, parse the base message to determine the type
	var base BaseMessage
	if err := json.Unmarshal([]byte(line), &base); err != nil {
		return fmt.Errorf("failed to parse base message: %w", err)
	}

	switch base.Type {
	case MessageTypeBoot:
		var msg BootMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			return fmt.Errorf("failed to parse boot message: %w", err)
		}
		handler.HandleBoot(portName, msg)

	case MessageTypeCard:
		var msg CardMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			return fmt.Errorf("failed to parse card message: %w", err)
		}
		handler.HandleCard(portName, msg)

	case MessageTypeError:
		var msg ErrorMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			return fmt.Errorf("failed to parse error message: %w", err)
		}
		handler.HandleError(portName, msg)

	default:
		slog.Warn("Unknown message type",
			slog.String("port", portName),
			slog.String("type", string(base.Type)),
			slog.String("line", line))
	}

	return nil
}
