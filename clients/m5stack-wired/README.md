# M5Stack Wired RFID Client

This is a USB serial-based RFID client for M5Stack ATOM devices that communicates with a Linux host via JSON Lines format over serial connection.

## Overview

Unlike the WiFi-based M5Stack client, this wired client:
- Communicates via **USB Serial** (115200 baud)
- Uses **JSON Lines** format (one JSON object per line)
- Requires **unique device_id** for each device
- Supports **multiple simultaneous devices** on a single Linux host

## Hardware Requirements

- M5Stack ATOM Lite or Matrix
- M5Stack RFID Reader (RC522-based)
- USB cable for connection to Linux host

## Message Format

### Boot Message
Sent on device startup:
```json
{"type":"boot","ts":"T+1234","device_id":"AA:BB:CC:DD:EE:FF","seq":1,"fw_version":"2811b6f","reason":"power_on"}
```

- `ts`: Timestamp in milliseconds since device boot (format: `T+{milliseconds}`)
- `device_id`: MAC address (if DEVICE_ID not specified) or custom device ID
- `seq`: Sequence number for this device (starts at 1, increments with each message)
- `fw_version`: Git commit hash of the firmware build
- `reason`: Boot reason (always "power_on" for this implementation)

### Card Message
Sent when RFID card is detected:
```json
{"type":"card","ts":"T+5678","device_id":"atom-door-01","seq":2,"card_uid":"04AABBCCDD11","tech":"MIFARE","rssi":0}
```

### Error Message
Sent when error occurs:
```json
{"type":"error","ts":"T+9012","device_id":"atom-door-01","seq":3,"code":"rfid_init_failed","message":"Failed to initialize RFID reader on channel 0"}
```

## Building and Flashing

### Prerequisites

- [uv](https://docs.astral.sh/uv/) (Python package manager)
- Python 3.13+

The project uses `uv` to manage the Python environment and PlatformIO dependencies.

### Build Configuration

The firmware accepts the following environment variables:

```bash
# Required: Client type (player, board, or muck)
export CLIENT_TYPE=player

# Optional: Unique device identifier (defaults to MAC address if not set)
export DEVICE_ID=atom-door-01
```

**Notes**:
- If `DEVICE_ID` is not specified, the device will use its MAC address as the device identifier, similar to the WiFi client.
- `FW_VERSION` is automatically set to the current git commit hash (e.g., `2811b6f`) during build time. No manual configuration needed.

### Setup Environment

First, set up the Python environment using `uv`:

```bash
cd m5stack-wired

# Install dependencies and create virtual environment
uv sync

# Activate the virtual environment (optional, uv run handles this automatically)
source .venv/bin/activate
```

### Build and Upload

```bash
# Basic build (uses MAC address as device_id)
uv run pio run -t upload --environment m5stack-atom -e CLIENT_TYPE=player

# With custom device ID
DEVICE_ID="atom-player-01" CLIENT_TYPE="player" uv run pio run -t upload --environment m5stack-atom

# For board mode (5 RFID readers)
CLIENT_TYPE="board" uv run pio run -t upload --environment m5stack-atom
```

**Alternative (without uv):**
```bash
# If virtual environment is activated
CLIENT_TYPE="player" pio run -t upload --environment m5stack-atom
```

## Client Types

- **player**: 2 RFID readers for hole cards, sends only when both cards detected
- **muck**: Same as player mode, for muck pile cards
- **board**: 5 RFID readers for community cards, sends each card immediately

## Device Identification

Each M5Stack device has a unique `device_id`. The server uses this to:
- Identify which physical reader generated an event
- Apply per-device debounce logic
- Enable monitoring and analytics

### Device ID Options

1. **Default (MAC Address)**: If no `DEVICE_ID` is specified, the device uses its MAC address (e.g., `AA:BB:CC:DD:EE:FF`)
   - Pros: Automatically unique, works immediately
   - Cons: Less human-readable

2. **Custom ID**: Set via build flag `DEVICE_ID` (e.g., `atom-entrance-01`)
   - Pros: Human-readable, easy to identify location
   - Cons: Must ensure uniqueness manually

### Recommended Custom Naming Scheme

```
atom-entrance-01
atom-entrance-02
atom-backdoor-01
atom-checkpoint-03
```

## Serial Port Settings

- **Baud rate**: 115200
- **Data bits**: 8
- **Parity**: None
- **Stop bits**: 1
- **Flow control**: None

## LED Indicator

- **Green**: Both cards detected (in player/muck mode)
- **Red**: Error occurred
- **Off**: Normal operation, waiting for cards

## Monitoring

To monitor the serial output:

```bash
# Using PlatformIO (recommended)
uv run pio device monitor --environment m5stack-atom --baud 115200

# Using screen
screen /dev/ttyUSB0 115200

# Using minicom
minicom -D /dev/ttyUSB0 -b 115200
```

Lines starting with `#` are debug comments and should be ignored by the server.
Lines starting with `{` are JSON Lines messages that should be parsed by the server.

## Troubleshooting

### Device not found
- Check USB cable connection
- Check `/dev/ttyUSB*` or `/dev/ttyACM*` device files
- Ensure proper permissions: `sudo chmod 666 /dev/ttyUSB0`

### RFID not working
- Check RFID reader I2C connection
- Check error messages in serial output
- Verify PaHub (TCA9548A) I2C multiplexer connection

### Cards not detected
- Ensure cards are placed close enough to the reader
- Check cooldown period (10 seconds between same card reads)
- Verify CLIENT_TYPE matches your hardware configuration

## Development

### Project Structure

```
m5stack-wired/
├── platformio.ini          # PlatformIO configuration
├── src/
│   ├── main.cpp           # Main application logic
│   ├── rfid.cpp           # RFID reading logic
│   └── serial_message.cpp # JSON Lines message formatting
└── README.md              # This file
```

### Adding New Message Types

1. Add message function in `serial_message.cpp`
2. Add forward declaration in relevant files
3. Call the function where needed

### Testing

You can test the serial output using a simple Python script:

```python
import serial
import json

ser = serial.Serial('/dev/ttyUSB0', 115200)

while True:
    line = ser.readline().decode('utf-8').strip()
    if line.startswith('{'):
        msg = json.loads(line)
        print(f"Received: {msg['type']} from {msg['device_id']}")
```

## License

Same as parent project.
