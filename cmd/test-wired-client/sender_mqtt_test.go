package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	mochi "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
)

// helper to start an in-process MQTT broker on a random port
func startTestBroker(t *testing.T) (*mochi.Server, int) {
	t.Helper()
	server := mochi.New(nil)
	_ = server.AddHook(new(auth.AllowHook), nil)

	// Use port 0 for auto-assignment, but mochi needs an explicit port.
	// Pick a high port and retry if needed.
	port := 18830 + (time.Now().UnixNano() % 100)
	addr := fmt.Sprintf(":%d", port)
	tcp := listeners.NewTCP(listeners.Config{
		ID:      "test",
		Address: addr,
	})
	err := server.AddListener(tcp)
	if err != nil {
		t.Fatalf("AddListener: %v", err)
	}
	go func() {
		_ = server.Serve()
	}()
	// Give the broker a moment to start
	time.Sleep(100 * time.Millisecond)
	return server, int(port)
}

func TestMQTTCardSender_SendCard(t *testing.T) {
	broker, port := startTestBroker(t)
	defer broker.Close()

	sender, err := NewMQTTCardSender(MQTTConfig{
		Broker:   "localhost",
		Port:     port,
		ClientID: "test-sender",
	})
	if err != nil {
		t.Fatalf("NewMQTTCardSender: %v", err)
	}
	defer sender.Close()

	// Subscribe with a separate client
	var received MQTTCardMessage
	var mu sync.Mutex
	done := make(chan struct{})

	subOpts := mqtt.NewClientOptions().
		AddBroker(fmt.Sprintf("tcp://localhost:%d", port)).
		SetClientID("test-sub")
	subClient := mqtt.NewClient(subOpts)
	token := subClient.Connect()
	token.Wait()
	if err := token.Error(); err != nil {
		t.Fatalf("sub connect: %v", err)
	}
	defer subClient.Disconnect(250)

	subClient.Subscribe("rfid-poker/device-01/card", 0, func(_ mqtt.Client, msg mqtt.Message) {
		mu.Lock()
		defer mu.Unlock()
		json.Unmarshal(msg.Payload(), &received)
		close(done)
	})

	time.Sleep(50 * time.Millisecond) // ensure subscription is active

	err = sender.SendCard(context.Background(), "04AABBCCDD11", "device-01", 1)
	if err != nil {
		t.Fatalf("SendCard: %v", err)
	}

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for message")
	}

	mu.Lock()
	defer mu.Unlock()
	if received.Type != "card" {
		t.Errorf("Type = %q, want card", received.Type)
	}
	if received.UID != "04AABBCCDD11" {
		t.Errorf("UID = %q, want 04AABBCCDD11", received.UID)
	}
	if received.DeviceID != "device-01" {
		t.Errorf("DeviceID = %q, want device-01", received.DeviceID)
	}
	if received.PairID != 1 {
		t.Errorf("PairID = %d, want 1", received.PairID)
	}
	if received.Seq != 1 {
		t.Errorf("Seq = %d, want 1", received.Seq)
	}
}

func TestMQTTCardSender_SendBoot(t *testing.T) {
	broker, port := startTestBroker(t)
	defer broker.Close()

	sender, err := NewMQTTCardSender(MQTTConfig{
		Broker:   "localhost",
		Port:     port,
		ClientID: "test-sender-boot",
	})
	if err != nil {
		t.Fatalf("NewMQTTCardSender: %v", err)
	}
	defer sender.Close()

	var received MQTTBootMessage
	var mu sync.Mutex
	done := make(chan struct{})

	subOpts := mqtt.NewClientOptions().
		AddBroker(fmt.Sprintf("tcp://localhost:%d", port)).
		SetClientID("test-sub-boot")
	subClient := mqtt.NewClient(subOpts)
	token := subClient.Connect()
	token.Wait()
	if err := token.Error(); err != nil {
		t.Fatalf("sub connect: %v", err)
	}
	defer subClient.Disconnect(250)

	subClient.Subscribe("rfid-poker/device-02/boot", 0, func(_ mqtt.Client, msg mqtt.Message) {
		mu.Lock()
		defer mu.Unlock()
		json.Unmarshal(msg.Payload(), &received)
		close(done)
	})

	time.Sleep(50 * time.Millisecond)

	err = sender.SendBoot(context.Background(), "device-02", []int{1, 2})
	if err != nil {
		t.Fatalf("SendBoot: %v", err)
	}

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for boot message")
	}

	mu.Lock()
	defer mu.Unlock()
	if received.Type != "boot" {
		t.Errorf("Type = %q, want boot", received.Type)
	}
	if received.DeviceID != "device-02" {
		t.Errorf("DeviceID = %q, want device-02", received.DeviceID)
	}
	if received.Reason != "test_client" {
		t.Errorf("Reason = %q, want test_client", received.Reason)
	}
	if len(received.PairIDs) != 2 || received.PairIDs[0] != 1 || received.PairIDs[1] != 2 {
		t.Errorf("PairIDs = %v, want [1, 2]", received.PairIDs)
	}
}

func TestMQTTCardSender_SequenceIncrement(t *testing.T) {
	broker, port := startTestBroker(t)
	defer broker.Close()

	sender, err := NewMQTTCardSender(MQTTConfig{
		Broker:   "localhost",
		Port:     port,
		ClientID: "test-sender-seq",
	})
	if err != nil {
		t.Fatalf("NewMQTTCardSender: %v", err)
	}
	defer sender.Close()

	var messages []MQTTCardMessage
	var mu sync.Mutex
	done := make(chan struct{})

	subOpts := mqtt.NewClientOptions().
		AddBroker(fmt.Sprintf("tcp://localhost:%d", port)).
		SetClientID("test-sub-seq")
	subClient := mqtt.NewClient(subOpts)
	token := subClient.Connect()
	token.Wait()
	if err := token.Error(); err != nil {
		t.Fatalf("sub connect: %v", err)
	}
	defer subClient.Disconnect(250)

	subClient.Subscribe("rfid-poker/device-seq/card", 0, func(_ mqtt.Client, msg mqtt.Message) {
		mu.Lock()
		defer mu.Unlock()
		var m MQTTCardMessage
		json.Unmarshal(msg.Payload(), &m)
		messages = append(messages, m)
		if len(messages) == 3 {
			close(done)
		}
	})

	time.Sleep(50 * time.Millisecond)

	for i := range 3 {
		err = sender.SendCard(context.Background(), fmt.Sprintf("uid-%d", i), "device-seq", 1)
		if err != nil {
			t.Fatalf("SendCard[%d]: %v", i, err)
		}
	}

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for messages")
	}

	mu.Lock()
	defer mu.Unlock()
	for i, m := range messages {
		expectedSeq := uint32(i + 1)
		if m.Seq != expectedSeq {
			t.Errorf("messages[%d].Seq = %d, want %d", i, m.Seq, expectedSeq)
		}
	}
}

func TestMQTTCardSender_Mode(t *testing.T) {
	broker, port := startTestBroker(t)
	defer broker.Close()

	sender, err := NewMQTTCardSender(MQTTConfig{
		Broker:   "localhost",
		Port:     port,
		ClientID: "test-sender-mode",
	})
	if err != nil {
		t.Fatalf("NewMQTTCardSender: %v", err)
	}
	defer sender.Close()

	if got := sender.Mode(); got != "MQTT" {
		t.Errorf("Mode() = %q, want MQTT", got)
	}
}
