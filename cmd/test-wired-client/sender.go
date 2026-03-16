package main

import "context"

// CardSender abstracts the transport layer for sending card/boot events.
type CardSender interface {
	SendCard(ctx context.Context, uid, deviceID string, pairID int) error
	SendBoot(ctx context.Context, deviceID string, pairIDs []int) error
	Close() error
	Mode() string // "HTTP" or "MQTT"
}

// MQTTCardMessage represents the JSON payload for card detection events.
type MQTTCardMessage struct {
	Type     string `json:"type"`
	TS       string `json:"ts"`
	DeviceID string `json:"device_id"`
	Seq      uint32 `json:"seq"`
	UID      string `json:"uid"`
	PairID   int    `json:"pair_id"`
}

// MQTTBootMessage represents the JSON payload for device boot events.
type MQTTBootMessage struct {
	Type      string `json:"type"`
	TS        string `json:"ts"`
	DeviceID  string `json:"device_id"`
	Seq       uint32 `json:"seq"`
	FwVersion string `json:"fw_version"`
	Reason    string `json:"reason"`
	PairIDs   []int  `json:"pair_ids"`
}
