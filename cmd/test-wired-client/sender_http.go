package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/goccy/go-json"
	"github.com/whywaita/rfid-poker/pkg/version"
)

// HTTPCardSender sends card/boot events via HTTP POST to the rfid-poker server.
type HTTPCardSender struct {
	client    *http.Client
	serverURL string
}

// NewHTTPCardSender creates a new HTTPCardSender.
func NewHTTPCardSender(serverURL string) *HTTPCardSender {
	return &HTTPCardSender{
		client:    &http.Client{Timeout: 5 * time.Second},
		serverURL: serverURL,
	}
}

func (s *HTTPCardSender) Mode() string { return "HTTP" }

func (s *HTTPCardSender) Close() error { return nil }

// PostCardRequest represents the request body for POST /card endpoint.
type PostCardRequest struct {
	UID      string `json:"uid"`
	DeviceID string `json:"device_id"`
	PairID   int    `json:"pair_id"`
}

func (s *HTTPCardSender) SendCard(ctx context.Context, uid, deviceID string, pairID int) error {
	req := PostCardRequest{
		UID:      uid,
		DeviceID: deviceID,
		PairID:   pairID,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.serverURL+"/card", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", fmt.Sprintf("test-wired-client/%s (%s)", version.GetVersion(), version.GetCommit()))

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusNotModified:
		return nil
	default:
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
	}
}

// Device represents the request body for POST /device/boot endpoint.
// This matches the server-side Device type in pkg/server/server_http_device.go.
type Device struct {
	DeviceID string `json:"device_id"`
	PairIDs  []int  `json:"pair_ids"`
}

func (s *HTTPCardSender) SendBoot(ctx context.Context, deviceID string, pairIDs []int) error {
	device := Device{
		DeviceID: deviceID,
		PairIDs:  pairIDs,
	}

	body, err := json.Marshal(device)
	if err != nil {
		return fmt.Errorf("failed to marshal boot request: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.serverURL+"/device/boot", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create boot request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", fmt.Sprintf("test-wired-client/%s (%s)", version.GetVersion(), version.GetCommit()))

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send boot request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated:
		return nil
	default:
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("boot status %d: %s", resp.StatusCode, string(respBody))
	}
}
