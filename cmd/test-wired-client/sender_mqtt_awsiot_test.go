//go:build manual

package main

import (
	"context"
	"fmt"
	"os"
	"testing"
)

// TestMQTTCardSender_AWSIoT is a manual integration test for AWS IoT Core.
// Run with: go test -tags manual -run TestMQTTCardSender_AWSIoT -v ./cmd/test-wired-client/
//
// Required env vars:
//
//	AWS_IOT_ENDPOINT - e.g. xxxx-ats.iot.ap-northeast-1.amazonaws.com
//	AWS_IOT_CA_CERT  - path to AmazonRootCA1.pem
//	AWS_IOT_CERT     - path to client certificate
//	AWS_IOT_KEY      - path to client private key
func TestMQTTCardSender_AWSIoT(t *testing.T) {
	endpoint := os.Getenv("AWS_IOT_ENDPOINT")
	caCert := os.Getenv("AWS_IOT_CA_CERT")
	cert := os.Getenv("AWS_IOT_CERT")
	key := os.Getenv("AWS_IOT_KEY")

	if endpoint == "" || caCert == "" || cert == "" || key == "" {
		t.Skip("AWS IoT env vars not set")
	}

	sender, err := NewMQTTCardSender(MQTTConfig{
		Broker:         endpoint,
		Port:           8883,
		ClientID:       fmt.Sprintf("abema-poker-test-go-%d", os.Getpid()),
		CACertFile:     caCert,
		ClientCertFile: cert,
		ClientKeyFile:  key,
		TopicPrefix:    "poker",
	})
	if err != nil {
		t.Fatalf("NewMQTTCardSender: %v", err)
	}
	defer sender.Close()

	t.Log("Connected to AWS IoT")

	ctx := context.Background()

	// Test SendCard
	err = sender.SendCard(ctx, "04AABBCCDD11", "test-device", 1)
	if err != nil {
		t.Fatalf("SendCard: %v", err)
	}
	t.Log("SendCard published successfully")

	// Test SendBoot
	err = sender.SendBoot(ctx, "test-device", []int{1})
	if err != nil {
		t.Fatalf("SendBoot: %v", err)
	}
	t.Log("SendBoot published successfully")
}
