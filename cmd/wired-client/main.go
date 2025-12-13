package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/whywaita/rfid-poker/pkg/serial"
	"github.com/whywaita/rfid-poker/pkg/version"
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
	var debug bool
	var noWatch bool

	flag.StringVar(&portPattern, "port", "", "Serial port pattern (e.g., /dev/ttyUSB0 or /dev/ttyUSB*)")
	flag.IntVar(&baudRate, "baud", serial.DefaultBaudRate, "Baud rate")
	flag.BoolVar(&listPorts, "list", false, "List available serial ports")
	flag.BoolVar(&noHTTPSend, "no-http-send", false, "Disable HTTP POST to server (console output only)")
	flag.StringVar(&serverURL, "server", "http://localhost:8080", "Server URL for HTTP POST")
	flag.BoolVar(&debug, "debug", false, "Enable debug logging (shows comment lines from device)")
	flag.BoolVar(&noWatch, "no-watch", false, "Disable dynamic USB hotplug support (use static port list)")
	flag.Parse()

	logLevel := slog.LevelInfo
	if debug {
		logLevel = slog.LevelDebug
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: false,
		Level:     logLevel,
	})))

	slog.Info("Starting wired-client",
		slog.String("version", version.GetVersion()),
		slog.String("commit", version.GetCommit()),
	)

	if listPorts {
		return listSerialPorts()
	}

	if portPattern == "" {
		return fmt.Errorf("port pattern is required. Use -port flag or -list to see available ports")
	}

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

	// Create context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		slog.Info("Received shutdown signal")
		cancel()
	}()

	if noWatch {
		// Static mode: read from existing ports only
		ports, err := serial.ExpandPortPattern(portPattern)
		if err != nil {
			return fmt.Errorf("failed to expand port pattern: %w", err)
		}

		if len(ports) == 0 {
			return fmt.Errorf("no serial ports found matching pattern: %s", portPattern)
		}

		slog.Info("Static mode: reading from existing ports only", slog.Int("count", len(ports)), slog.Any("ports", ports))
		return reader.ReadPorts(ctx, ports)
	}

	// Default: Dynamic hotplug mode - watch for device connect/disconnect
	slog.Info("Watch mode enabled: monitoring for USB hotplug events")
	return reader.WatchAndReadPorts(ctx, portPattern)
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
