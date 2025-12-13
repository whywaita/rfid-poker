package serial

// MessageType represents the type of message from M5Stack wired client
type MessageType string

const (
	MessageTypeBoot  MessageType = "boot"
	MessageTypeCard  MessageType = "card"
	MessageTypeError MessageType = "error"
)

// BaseMessage contains common fields for all messages
type BaseMessage struct {
	Type     MessageType `json:"type"`
	TS       string      `json:"ts"`
	DeviceID string      `json:"device_id"`
	Seq      int64       `json:"seq"`
}

// BootMessage is sent when the device boots
type BootMessage struct {
	BaseMessage
	FwVersion string `json:"fw_version"`
	Reason    string `json:"reason"`
}

// CardMessage is sent when a card is detected
type CardMessage struct {
	BaseMessage
	CardUID string `json:"card_uid"`
	Tech    string `json:"tech"`
	RSSI    int    `json:"rssi"`
}

// ErrorMessage is sent when an error occurs
type ErrorMessage struct {
	BaseMessage
	Code    string `json:"code"`
	Message string `json:"message"`
}
