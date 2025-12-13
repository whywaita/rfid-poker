package serial

import (
	"fmt"
	"log/slog"
)

// ConsoleMessageHandler implements MessageHandler for console output
type ConsoleMessageHandler struct{}

// NewConsoleMessageHandler creates a new ConsoleMessageHandler
func NewConsoleMessageHandler() *ConsoleMessageHandler {
	return &ConsoleMessageHandler{}
}

func (h *ConsoleMessageHandler) HandleBoot(portName string, msg BootMessage) {
	fmt.Printf("[%s] 🟢 BOOT [seq=%d, ts=%s]\n", portName, msg.Seq, msg.TS)
	fmt.Printf("[%s]    Device ID: %s\n", portName, msg.DeviceID)
	fmt.Printf("[%s]    Firmware:  %s\n", portName, msg.FwVersion)
	fmt.Printf("[%s]    Reason:    %s\n", portName, msg.Reason)
	fmt.Println("---")

	slog.Info("Device booted",
		slog.String("port", portName),
		slog.String("device_id", msg.DeviceID),
		slog.String("fw_version", msg.FwVersion),
		slog.String("reason", msg.Reason),
		slog.Int64("seq", msg.Seq),
	)
}

func (h *ConsoleMessageHandler) HandleCard(portName string, msg CardMessage) {
	fmt.Printf("[%s] 🃏 CARD [seq=%d, ts=%s]\n", portName, msg.Seq, msg.TS)
	fmt.Printf("[%s]    Device ID: %s\n", portName, msg.DeviceID)
	fmt.Printf("[%s]    Card UID:  %s\n", portName, msg.CardUID)
	fmt.Printf("[%s]    Tech:      %s\n", portName, msg.Tech)
	fmt.Printf("[%s]    RSSI:      %d\n", portName, msg.RSSI)
	fmt.Println("---")

	slog.Info("Card detected",
		slog.String("port", portName),
		slog.String("device_id", msg.DeviceID),
		slog.String("card_uid", msg.CardUID),
		slog.String("tech", msg.Tech),
		slog.Int("rssi", msg.RSSI),
		slog.Int64("seq", msg.Seq),
	)
}

func (h *ConsoleMessageHandler) HandleError(portName string, msg ErrorMessage) {
	fmt.Printf("[%s] ❌ ERROR [seq=%d, ts=%s]\n", portName, msg.Seq, msg.TS)
	fmt.Printf("[%s]    Device ID: %s\n", portName, msg.DeviceID)
	fmt.Printf("[%s]    Code:      %s\n", portName, msg.Code)
	fmt.Printf("[%s]    Message:   %s\n", portName, msg.Message)
	fmt.Println("---")

	slog.Error("Device error",
		slog.String("port", portName),
		slog.String("device_id", msg.DeviceID),
		slog.String("code", msg.Code),
		slog.String("message", msg.Message),
		slog.Int64("seq", msg.Seq),
	)
}
