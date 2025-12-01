package serial

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/goccy/go-json"
)

const (
	// NotModifiedBufferDuration is the duration to skip sending the same card
	// after receiving a 304 Not Modified response
	NotModifiedBufferDuration = 10 * time.Second
)

// HTTPMessageHandler implements MessageHandler for sending messages to HTTP server
type HTTPMessageHandler struct {
	serverURL  string
	httpClient *http.Client

	// cardBuffer stores the time until which a card should be skipped, keyed by "deviceID:cardUID"
	cardBuffer   map[string]time.Time
	cardBufferMu sync.RWMutex
}

// NewHTTPMessageHandler creates a new HTTPMessageHandler
func NewHTTPMessageHandler(serverURL string) *HTTPMessageHandler {
	return &HTTPMessageHandler{
		serverURL: serverURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		cardBuffer: make(map[string]time.Time),
	}
}

// shouldSkipCard checks if the card should be skipped based on the buffer
func (h *HTTPMessageHandler) shouldSkipCard(deviceID, cardUID string) bool {
	key := deviceID + ":" + cardUID

	h.cardBufferMu.RLock()
	defer h.cardBufferMu.RUnlock()

	skipUntil, exists := h.cardBuffer[key]
	if !exists {
		return false
	}

	return time.Now().Before(skipUntil)
}

// bufferCard marks the card to be skipped for the specified duration
func (h *HTTPMessageHandler) bufferCard(deviceID, cardUID string, duration time.Duration) {
	key := deviceID + ":" + cardUID

	h.cardBufferMu.Lock()
	defer h.cardBufferMu.Unlock()

	h.cardBuffer[key] = time.Now().Add(duration)
}

// PostCardRequest represents the request body for POST /card endpoint
type PostCardRequest struct {
	UID      string `json:"uid"`
	DeviceID string `json:"device_id"`
	PairID   int    `json:"pair_id"`
}

func (h *HTTPMessageHandler) HandleBoot(portName string, msg BootMessage) {
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

func (h *HTTPMessageHandler) HandleCard(portName string, msg CardMessage) {
	fmt.Printf("[%s] 🃏 CARD [seq=%d, ts=%s]\n", portName, msg.Seq, msg.TS)
	fmt.Printf("[%s]    Device ID: %s\n", portName, msg.DeviceID)
	fmt.Printf("[%s]    Card UID:  %s\n", portName, msg.CardUID)
	fmt.Printf("[%s]    Tech:      %s\n", portName, msg.Tech)
	fmt.Printf("[%s]    RSSI:      %d\n", portName, msg.RSSI)

	cardUID := strings.TrimSpace(msg.CardUID)

	// Check if this card should be skipped (buffered due to 304 response)
	if h.shouldSkipCard(msg.DeviceID, cardUID) {
		fmt.Printf("[%s]    ⏭️  Skipped (buffered)\n", portName)
		fmt.Println("---")
		slog.Debug("Card skipped (buffered)",
			slog.String("port", portName),
			slog.String("device_id", msg.DeviceID),
			slog.String("card_uid", cardUID),
		)
		return
	}

	// Extract pair_id from device_id if it contains a hyphen
	// Expected format: "device_id-pair_id" or just "device_id"
	deviceID := msg.DeviceID
	pairID := 1 // Default pair_id

	// Try to extract pair_id from device_id
	if idx := strings.LastIndex(deviceID, "-"); idx != -1 {
		pairIDStr := deviceID[idx+1:]
		if pid, err := strconv.Atoi(pairIDStr); err == nil {
			pairID = pid
			deviceID = deviceID[:idx] // Remove pair_id from device_id
		}
	}

	// Send to HTTP server
	req := PostCardRequest{
		UID:      cardUID,
		DeviceID: deviceID,
		PairID:   pairID,
	}

	go func() {
		// send HTTP request in a separate goroutine
		statusCode, err := h.postCard(req)
		if err != nil {
			slog.Error("Failed to send card to HTTP server",
				slog.String("port", portName),
				slog.String("device_id", msg.DeviceID),
				slog.String("card_uid", cardUID),
				slog.String("error", err.Error()),
			)
			fmt.Printf("[%s]    ❌ Failed to send to server: %v\n", portName, err)
			return
		}

		if statusCode == http.StatusNotModified {
			// Card already exists on server, buffer it
			h.bufferCard(msg.DeviceID, cardUID, NotModifiedBufferDuration)
			fmt.Printf("[%s]    📋 Card already registered (304), buffering for %v\n", portName, NotModifiedBufferDuration)
			slog.Info("Card already registered, buffering",
				slog.String("port", portName),
				slog.String("device_id", msg.DeviceID),
				slog.String("card_uid", cardUID),
				slog.Duration("buffer_duration", NotModifiedBufferDuration),
			)
		} else {
			fmt.Printf("[%s]    ✅ Sent to server: %s (pair_id=%d)\n", portName, h.serverURL, pairID)
		}
	}()

	fmt.Println("---")

	slog.Info("Card detected",
		slog.String("port", portName),
		slog.String("device_id", msg.DeviceID),
		slog.String("card_uid", cardUID),
		slog.String("tech", msg.Tech),
		slog.Int("rssi", msg.RSSI),
		slog.Int64("seq", msg.Seq),
	)
}

func (h *HTTPMessageHandler) HandleError(portName string, msg ErrorMessage) {
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

func (h *HTTPMessageHandler) postCard(req PostCardRequest) (int, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", h.serverURL+"/card", bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(httpReq)
	if err != nil {
		return 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Accept 200, 201, and 304 as valid responses
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNotModified {
		return resp.StatusCode, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return resp.StatusCode, nil
}
