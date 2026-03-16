package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/goccy/go-json"

	"github.com/whywaita/rfid-poker/pkg/version"
)

// MQTTCardSender sends card/boot events via MQTT publish.
type MQTTCardSender struct {
	client      mqtt.Client
	seq         atomic.Uint32
	startTime   time.Time
	topicPrefix string
}

// MQTTConfig holds MQTT connection parameters.
type MQTTConfig struct {
	Broker   string
	Port     int
	User     string
	Password string
	ClientID string

	// TLS settings (for AWS IoT Core etc.)
	CACertFile     string // path to CA certificate
	ClientCertFile string // path to client certificate
	ClientKeyFile  string // path to client private key

	// Topic prefix override (default: "rfid-poker")
	TopicPrefix string
}

// NewMQTTCardSender creates a new MQTTCardSender and connects to the broker.
func NewMQTTCardSender(cfg MQTTConfig) (*MQTTCardSender, error) {
	scheme := "tcp"
	if cfg.CACertFile != "" {
		scheme = "tls"
	}

	opts := mqtt.NewClientOptions().
		AddBroker(fmt.Sprintf("%s://%s:%d", scheme, cfg.Broker, cfg.Port)).
		SetClientID(cfg.ClientID).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(5 * time.Second).
		SetKeepAlive(15 * time.Second)

	if cfg.User != "" {
		opts.SetUsername(cfg.User)
		opts.SetPassword(cfg.Password)
	}

	if cfg.CACertFile != "" {
		tlsConfig, err := newTLSConfig(cfg.CACertFile, cfg.ClientCertFile, cfg.ClientKeyFile)
		if err != nil {
			return nil, fmt.Errorf("tls config: %w", err)
		}
		opts.SetTLSConfig(tlsConfig)
	}

	topicPrefix := cfg.TopicPrefix
	if topicPrefix == "" {
		topicPrefix = "rfid-poker"
	}

	client := mqtt.NewClient(opts)
	token := client.Connect()
	token.Wait()
	if err := token.Error(); err != nil {
		return nil, fmt.Errorf("mqtt connect: %w", err)
	}

	return &MQTTCardSender{
		client:      client,
		startTime:   time.Now(),
		topicPrefix: topicPrefix,
	}, nil
}

func newTLSConfig(caFile, certFile, keyFile string) (*tls.Config, error) {
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read CA cert: %w", err)
	}
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA cert")
	}

	tlsConfig := &tls.Config{
		RootCAs:    caCertPool,
		MinVersion: tls.VersionTLS12,
	}

	if certFile != "" && keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("load client cert: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
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

	topic := fmt.Sprintf("%s/%s/card", s.topicPrefix, deviceID)
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

	topic := fmt.Sprintf("%s/%s/boot", s.topicPrefix, deviceID)
	token := s.client.Publish(topic, 0, false, payload)
	token.Wait()
	return token.Error()
}
