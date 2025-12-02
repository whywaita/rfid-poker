package serial

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	bugstserial "go.bug.st/serial"
)

const (
	DefaultBaudRate    = 115200
	RetryInterval      = 2 * time.Second
	PortStabilizeDelay = 500 * time.Millisecond
)

// Reader manages reading from serial ports
type Reader struct {
	baudRate int
	handler  MessageHandler

	// For dynamic port management
	mu          sync.Mutex
	activePorts map[string]context.CancelFunc
	portPattern string
}

// NewReader creates a new serial reader
func NewReader(baudRate int, handler MessageHandler) *Reader {
	if baudRate == 0 {
		baudRate = DefaultBaudRate
	}
	return &Reader{
		baudRate:    baudRate,
		handler:     handler,
		activePorts: make(map[string]context.CancelFunc),
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

// WatchAndReadPorts watches for serial port changes and reads from matching ports dynamically
func (r *Reader) WatchAndReadPorts(ctx context.Context, pattern string) error {
	r.portPattern = pattern

	// Extract the directory to watch from the pattern
	watchDir := filepath.Dir(pattern)
	if watchDir == "" {
		watchDir = "/dev"
	}

	// Create fsnotify watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer watcher.Close()

	// Start watching the directory
	if err := watcher.Add(watchDir); err != nil {
		return fmt.Errorf("failed to watch directory %s: %w", watchDir, err)
	}
	slog.Info("Watching for device changes", slog.String("directory", watchDir), slog.String("pattern", pattern))

	// Start reading from existing ports
	existingPorts, err := ExpandPortPattern(pattern)
	if err != nil {
		slog.Warn("Failed to expand port pattern", slog.String("error", err.Error()))
	} else {
		for _, port := range existingPorts {
			r.startPortReader(ctx, port)
		}
	}

	// Watch for new devices
	for {
		select {
		case <-ctx.Done():
			slog.Info("Context cancelled, stopping port watcher")
			r.stopAllPorts()
			return ctx.Err()

		case event, ok := <-watcher.Events:
			if !ok {
				return fmt.Errorf("watcher channel closed")
			}

			if event.Op&fsnotify.Create != 0 {
				// New device connected
				if r.matchesPattern(event.Name) {
					slog.Info("New device detected", slog.String("port", event.Name))
					// Wait a bit for the device to stabilize
					time.Sleep(PortStabilizeDelay)
					r.startPortReader(ctx, event.Name)
				}
			} else if event.Op&fsnotify.Remove != 0 {
				// Device removed
				if r.isActivePort(event.Name) {
					slog.Info("Device removed", slog.String("port", event.Name))
					r.stopPort(event.Name)
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return fmt.Errorf("watcher error channel closed")
			}
			slog.Warn("Watcher error", slog.String("error", err.Error()))
		}
	}
}

// matchesPattern checks if a port path matches the configured pattern
func (r *Reader) matchesPattern(portPath string) bool {
	matched, err := filepath.Match(r.portPattern, portPath)
	if err != nil {
		return false
	}
	return matched
}

// isActivePort checks if a port is currently being read
func (r *Reader) isActivePort(portPath string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, exists := r.activePorts[portPath]
	return exists
}

// startPortReader starts a goroutine to read from a port
func (r *Reader) startPortReader(ctx context.Context, portPath string) {
	r.mu.Lock()
	if _, exists := r.activePorts[portPath]; exists {
		r.mu.Unlock()
		slog.Debug("Port reader already running", slog.String("port", portPath))
		return
	}

	portCtx, cancel := context.WithCancel(ctx)
	r.activePorts[portPath] = cancel
	r.mu.Unlock()

	go func() {
		defer func() {
			r.mu.Lock()
			delete(r.activePorts, portPath)
			r.mu.Unlock()
		}()

		r.ReadPortWithRetry(portCtx, portPath)
	}()
}

// stopPort stops reading from a specific port
func (r *Reader) stopPort(portPath string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if cancel, exists := r.activePorts[portPath]; exists {
		cancel()
		delete(r.activePorts, portPath)
	}
}

// stopAllPorts stops all active port readers
func (r *Reader) stopAllPorts() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for portPath, cancel := range r.activePorts {
		slog.Info("Stopping port reader", slog.String("port", portPath))
		cancel()
	}
	r.activePorts = make(map[string]context.CancelFunc)
}

// ReadPortWithRetry reads from a port with automatic retry on disconnection
func (r *Reader) ReadPortWithRetry(ctx context.Context, portName string) {
	for {
		err := r.ReadPort(ctx, portName)

		// Check if context was cancelled (intentional shutdown)
		if ctx.Err() != nil {
			slog.Info("Port reader stopped", slog.String("port", portName))
			return
		}

		// Check if the port still exists
		if _, statErr := os.Stat(portName); os.IsNotExist(statErr) {
			slog.Info("Port no longer exists, stopping retry", slog.String("port", portName))
			return
		}

		slog.Warn("Port disconnected, will retry",
			slog.String("port", portName),
			slog.String("error", err.Error()),
			slog.Duration("retry_in", RetryInterval))

		select {
		case <-ctx.Done():
			return
		case <-time.After(RetryInterval):
			slog.Info("Retrying port connection", slog.String("port", portName))
		}
	}
}
