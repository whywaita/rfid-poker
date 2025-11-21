# RFID Client Requirements (rdd.md)

This document defines the specification requirements for the **client devices**, consisting of multiple M5Stack ATOM units equipped with RFID readers.  
Each device connects to a Linux host via USB serial, and all devices collectively feed events into a Go-based HTTP server running on the host.

The system is designed to support **multiple M5Stack units operating simultaneously**, each uniquely identified and behaving independently while conforming to a common communication protocol.

---

## 1. System Overview

- **Hardware**
  - Multiple M5Stack ATOM devices (Lite / Matrix)
  - Multiple M5Stack RFID Readers (one per ATOM)
  - Linux host capable of handling multiple serial ports
- **Communication**
  - Each ATOM connects via USB serial to the Linux host  
    (Linux may expose `/dev/ttyUSB0`, `/dev/ttyUSB1`, `/dev/ttyUSB2`, etc.)
  - Communication format: **JSON Lines** (one JSON object per line)
- **Multiplexing Strategy**
  - Each device emits messages independently
  - The server:
    - Opens and continuously reads from multiple serial device paths
    - Merges and processes events from all devices
    - Identifies devices based on the `device_id` field embedded in messages
- **Responsibilities**
  - **Client (M5Stack)**:
    - Read RFID data
    - Emit structured JSON messages via serial
    - Issue a boot message upon startup
    - Emit error messages when failures occur
    - Provide a unique `device_id` for identification
  - **Server (Go)**:
    - Maintain multiple serial readers (one per device)
    - Parse JSON messages
    - Apply debounce logic
    - Persist and process card events across all devices

---

## 2. Multi-Device Serial Communication Specification

### 2.1 Physical Layer

- The Linux host will have multiple serial devices:
  - Example: `/dev/ttyUSB0`, `/dev/ttyUSB1`, `/dev/ttyUSB2`, ...
- Each M5 device must behave identically and independently.

### 2.2 Serial Port Settings

All devices use the same serial port configuration:

| Parameter     | Value  |
|---------------|--------|
| Baud rate     | 115200 |
| Data bits     | 8      |
| Parity        | None   |
| Stop bits     | 1      |
| Flow control  | None   |

### 2.3 Message Framing

- One message per line
- Line ends with `\n`
- The Linux Go server will run multiple reader goroutines, one per serial port

---

## 3. Multi-Device Message Format Requirements

### 3.1 Common Fields Across All Devices

All messages from all devices must include:

| Field        | Type    | Required | Description                                                            |
|--------------|---------|----------|------------------------------------------------------------------------|
| `type`       | string  | ✔        | `"boot"`, `"card"`, `"error"`                                          |
| `ts`         | string  | ✔        | Timestamp in ISO8601 format                                            |
| `device_id`  | string  | ✔        | **Unique identifier per physical ATOM device** (e.g., `"atom-door-01"`) |
| `seq`        | number  | ✔        | Sequence number since boot                                             |

### 3.2 Device Identification Requirement

- Each M5Stack must possess a **unique, hard-coded or configurable** `device_id`.
- Example naming scheme (recommended):

```

atom-entrance-01
atom-entrance-02
atom-backdoor-01
atom-checkpoint-03

````

- The server relies on the `device_id` field for:
  - Identifying which physical reader generated an event
  - Applying per-device debounce
  - Logging, monitoring, and analytics
- Device IDs MUST NOT collide.

---

## 4. Message Types

(Identical to the base specification, with emphasis on multi-device operation.)

---

### 4.1 Boot Message (`type = "boot"`)

Sent by every device on startup.

```json
{
  "type": "boot",
  "ts": "2025-11-20T12:34:56Z",
  "device_id": "atom-door-01",
  "seq": 1,
  "fw_version": "1.0.0",
  "reason": "power_on"
}
````

### 4.1.1 Purpose in Multi-Device Environment

* Allows the server to detect:

  * Device restarts
  * Device connection ordering
  * Newly added or replaced devices

---

### 4.2 Card Message (`type = "card"`)

```json
{
  "type": "card",
  "ts": "2025-11-20T12:35:10Z",
  "device_id": "atom-door-01",
  "seq": 42,
  "card_uid": "04AABBCCDD11",
  "tech": "MIFARE",
  "rssi": -60
}
```

### 4.2.1 Multi-Device Considerations

* The server performs debounce per `device_id`.
* Events from all devices share the same logical global stream.
* Device position/location can be derived from `device_id`.

---

### 4.3 Error Message (`type = "error"`)

```json
{
  "type": "error",
  "ts": "2025-11-20T12:36:00Z",
  "device_id": "atom-door-01",
  "seq": 100,
  "code": "rfid_init_failed",
  "message": "RC522 init timeout"
}
```

### 4.3.1 Multi-Device Considerations

* Enables centralized monitoring:
  “Is device X having trouble?”
  “Did device Y disconnect or fail initialization?”

---

## 5. Example Combined Log (Multi-Device)

```
{"type":"boot","ts":"2025-11-20T12:34:56Z","device_id":"atom-door-01","seq":1,"fw_version":"1.0.0","reason":"power_on"}
{"type":"boot","ts":"2025-11-20T12:35:01Z","device_id":"atom-door-02","seq":1,"fw_version":"1.0.0","reason":"power_on"}

{"type":"card","ts":"2025-11-20T12:35:10Z","device_id":"atom-door-01","seq":2,"card_uid":"04AABBCCDD11"}
{"type":"card","ts":"2025-11-20T12:35:12Z","device_id":"atom-door-02","seq":2,"card_uid":"04FFEEDDCC99"}
{"type":"card","ts":"2025-11-20T12:35:15Z","device_id":"atom-door-01","seq":3,"card_uid":"04AABBCCDD11"}

{"type":"error","ts":"2025-11-20T12:36:00Z","device_id":"atom-door-02","seq":4,"code":"rfid_read_error","message":"CRC mismatch"}
```

---

## 6. Multi-Device Implementation Notes (Client Side)

### 6.1 Device ID Management

Each device must have:

* A **unique** `device_id`
* Configurable via:

  * Hardcoded build flag
  * Flash-stored configuration
  * DIP switch / serial config tool (optional)

### 6.2 Independence

Each M5:

* Boots independently
* Maintains its own `seq` counter
* Emits messages without knowing about other devices
* Does NOT coordinate with peers

### 6.3 Server-side Multiplexing Expectation

The server:

* Opens multiple serial ports concurrently
* Runs one reader goroutine per port
* Parses messages from all devices into a unified event stream

### 6.4 Error Reporting Benefits

* Alerts the server to malfunctioning units
* Enables health dashboards or alerting systems

---

## 7. Non-Functional Requirements (Multi-Device)

* **Scalability**

  * System must support multiple ATOM devices connected simultaneously
  * No assumptions about port order or stability of `/dev/ttyUSB*` mapping
* **Extensibility**

  * Additional devices may be added without changing other devices' behavior
* **Fault Isolation**

  * A malfunction in one device must not affect others
* **Consistency**

  * All devices must follow the same message schema and behavior

---

## 8. Future Multi-Device Extensions (Optional)

* Device discovery via periodic `"heartbeat"` messages
* Dynamic assignment of device IDs
* Automatic mapping between USB port and device ID
* Centralized firmware update system
* Hot-plug detection and automated registration on the host

