package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/whywaita/rfid-poker/pkg/serial"
)

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	var portPattern string
	var baudRate int
	var listPorts bool
	var noHTTPSend bool
	var serverURL string

	flag.StringVar(&portPattern, "port", "", "Serial port pattern (e.g., /dev/ttyUSB0 or /dev/ttyUSB*)")
	flag.IntVar(&baudRate, "baud", serial.DefaultBaudRate, "Baud rate")
	flag.BoolVar(&listPorts, "list", false, "List available serial ports")
	flag.BoolVar(&noHTTPSend, "no-http-send", false, "Disable HTTP POST to server (console output only)")
	flag.StringVar(&serverURL, "server", "http://localhost:8080", "Server URL for HTTP POST")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: false,
		Level:     slog.LevelInfo,
	})))

	if listPorts {
		return listSerialPorts()
	}

	if portPattern == "" {
		return fmt.Errorf("port pattern is required. Use -port flag or -list to see available ports")
	}

	// Expand glob pattern
	ports, err := serial.ExpandPortPattern(portPattern)
	if err != nil {
		return fmt.Errorf("failed to expand port pattern: %w", err)
	}

	if len(ports) == 0 {
		return fmt.Errorf("no serial ports found matching pattern: %s", portPattern)
	}

	slog.Info("Found serial ports", slog.Int("count", len(ports)), slog.Any("ports", ports))

	// Create message handler based on flags
	var handler serial.MessageHandler
	if noHTTPSend {
		slog.Info("Using console-only mode (no HTTP POST)")
		handler = serial.NewConsoleMessageHandler()
	} else {
		slog.Info("Using HTTP POST mode", slog.String("server", serverURL))
		handler = serial.NewHTTPMessageHandler(serverURL)
	}

	// Create reader
	reader := serial.NewReader(baudRate, handler)

	// Read from multiple ports concurrently
	ctx := context.Background()
	return reader.ReadPorts(ctx, ports)
}

func listSerialPorts() error {
	ports, err := serial.ListSerialPorts()
	if err != nil {
		return fmt.Errorf("failed to list serial ports: %w", err)
	}

	if len(ports) == 0 {
		fmt.Println("No serial ports found")
		return nil
	}

	fmt.Println("Available serial ports:")
	for _, port := range ports {
		fmt.Printf("  %s\n", port)
	}
	return nil
}
