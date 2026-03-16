package main

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/goccy/go-json"

	"github.com/whywaita/rfid-poker/pkg/version"
)

// MQTTCardSender sends card/boot events via MQTT publish.
type MQTTCardSender struct {
	client    mqtt.Client
	seq       atomic.Uint32
	startTime time.Time
}

// MQTTConfig holds MQTT connection parameters.
type MQTTConfig struct {
	Broker   string
	Port     int
	User     string
	Password string
	ClientID string
}

// NewMQTTCardSender creates a new MQTTCardSender and connects to the broker.
func NewMQTTCardSender(cfg MQTTConfig) (*MQTTCardSender, error) {
	opts := mqtt.NewClientOptions().
		AddBroker(fmt.Sprintf("tcp://%s:%d", cfg.Broker, cfg.Port)).
		SetClientID(cfg.ClientID).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(5 * time.Second).
		SetKeepAlive(15 * time.Second)

	if cfg.User != "" {
		opts.SetUsername(cfg.User)
		opts.SetPassword(cfg.Password)
	}

	client := mqtt.NewClient(opts)
	token := client.Connect()
	token.Wait()
	if err := token.Error(); err != nil {
		return nil, fmt.Errorf("mqtt connect: %w", err)
	}

	return &MQTTCardSender{
		client:    client,
		startTime: time.Now(),
	}, nil
}

func (s *MQTTCardSender) Mode() string { return "MQTT" }

func (s *MQTTCardSender) Close() error {
	s.client.Disconnect(250)
	return nil
}

func (s *MQTTCardSender) SendCard(ctx context.Context, uid, deviceID string, pairID int) error {
	msg := MQTTCardMessage{
		Type:     "card",
		TS:       fmt.Sprintf("T+%d", time.Since(s.startTime).Milliseconds()),
		DeviceID: deviceID,
		Seq:      s.seq.Add(1),
		UID:      uid,
		PairID:   pairID,
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal card message: %w", err)
	}

	topic := fmt.Sprintf("rfid-poker/%s/card", deviceID)
	token := s.client.Publish(topic, 0, false, payload)
	token.Wait()
	return token.Error()
}

func (s *MQTTCardSender) SendBoot(ctx context.Context, deviceID string, pairIDs []int) error {
	msg := MQTTBootMessage{
		Type:      "boot",
		TS:        fmt.Sprintf("T+%d", time.Since(s.startTime).Milliseconds()),
		DeviceID:  deviceID,
		Seq:       s.seq.Add(1),
		FwVersion: version.GetVersion(),
		Reason:    "test_client",
		PairIDs:   pairIDs,
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal boot message: %w", err)
	}

	topic := fmt.Sprintf("rfid-poker/%s/boot", deviceID)
	token := s.client.Publish(topic, 0, false, payload)
	token.Wait()
	return token.Error()
}
