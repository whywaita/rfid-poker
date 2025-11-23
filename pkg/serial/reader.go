package serial

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	bugstserial "go.bug.st/serial"
)

const (
	DefaultBaudRate = 115200
)

// Reader manages reading from serial ports
type Reader struct {
	baudRate int
	handler  MessageHandler
}

// NewReader creates a new serial reader
func NewReader(baudRate int, handler MessageHandler) *Reader {
	if baudRate == 0 {
		baudRate = DefaultBaudRate
	}
	return &Reader{
		baudRate: baudRate,
		handler:  handler,
	}
}

// ExpandPortPattern expands a glob pattern to a list of serial port paths
func ExpandPortPattern(pattern string) ([]string, error) {
	// Check if pattern contains glob characters
	if !containsGlobChars(pattern) {
		// No glob pattern, return as-is if the port exists
		if _, err := os.Stat(pattern); err == nil {
			return []string{pattern}, nil
		}
		return nil, fmt.Errorf("port not found: %s", pattern)
	}

	// Expand glob pattern
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern: %w", err)
	}

	// Filter to only existing device files
	var ports []string
	for _, match := range matches {
		if info, err := os.Stat(match); err == nil && !info.IsDir() {
			ports = append(ports, match)
		}
	}

	return ports, nil
}

func containsGlobChars(s string) bool {
	for _, c := range s {
		if c == '*' || c == '?' || c == '[' || c == ']' {
			return true
		}
	}
	return false
}

// ReadPorts reads from multiple serial ports concurrently
func (r *Reader) ReadPorts(ctx context.Context, ports []string) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(ports))

	for _, portName := range ports {
		wg.Add(1)
		go func(port string) {
			defer wg.Done()
			if err := r.ReadPort(ctx, port); err != nil {
				errChan <- fmt.Errorf("port %s: %w", port, err)
			}
		}(portName)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(errChan)

	// Collect errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors occurred: %v", errs)
	}

	return nil
}

// ReadPort reads from a single serial port
func (r *Reader) ReadPort(ctx context.Context, portName string) error {
	slog.Info("Opening serial port", slog.String("port", portName), slog.Int("baud", r.baudRate))

	mode := &bugstserial.Mode{
		BaudRate: r.baudRate,
	}

	port, err := bugstserial.Open(portName, mode)
	if err != nil {
		return fmt.Errorf("failed to open serial port: %w", err)
	}
	defer port.Close()

	slog.Info("Serial port opened successfully", slog.String("port", portName))

	scanner := bufio.NewScanner(port)
	for {
		select {
		case <-ctx.Done():
			slog.Info("Context cancelled, closing serial port", slog.String("port", portName))
			return ctx.Err()
		default:
			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					return fmt.Errorf("error reading from serial port: %w", err)
				}
				return nil
			}

			line := scanner.Text()

			// Skip empty lines
			if line == "" {
				continue
			}

			// Skip comment lines (lines starting with #)
			if len(line) > 0 && line[0] == '#' {
				slog.Debug("Comment line", slog.String("port", portName), slog.String("line", line))
				continue
			}

			// Try to parse as JSON
			if err := ParseAndHandle(portName, line, r.handler); err != nil {
				slog.Warn("Failed to handle message",
					slog.String("port", portName),
					slog.String("error", err.Error()),
					slog.String("line", line))
			}
		}
	}
}

// ListSerialPorts returns a list of available serial ports
func ListSerialPorts() ([]string, error) {
	ports, err := bugstserial.GetPortsList()
	if err != nil {
		return nil, fmt.Errorf("failed to get serial ports: %w", err)
	}
	return ports, nil
}
