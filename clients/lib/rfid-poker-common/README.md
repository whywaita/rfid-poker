# RFID Poker Common Library

Shared RFID reading logic and card handling abstractions for M5Stack-based poker card readers.

## Overview

This library provides common functionality for both WiFi-based (`m5stack`) and USB Serial-based (`m5stack-wired`) RFID clients. It eliminates code duplication and provides a clean abstraction for card detection and processing.

## Architecture

### Core Components

#### 1. `CardHandler` (Abstract Interface)
Defines how card events are handled (HTTP, Serial, etc.)

```cpp
class CardHandler {
  virtual void onCardDetected(int channel, const char* uid) = 0;
  virtual void onError(const char* code, const char* message) = 0;
  virtual void onBoot(const char* reason) = 0;
};
```

**Implementations:**
- `HttpCardHandler` (in `m5stack`): Sends data via HTTP POST
- `SerialCardHandler` (in `m5stack-wired`): Sends JSON Lines via Serial

#### 2. `RfidCore`
Core RFID reading logic shared across all clients

**Features:**
- Multi-reader support (up to 6 RFID readers)
- Automatic channel multiplexing via TCA9548A
- Card detection with 30-second cooldown
- Client type support (player, board, muck)
- Pair completion detection

**API:**
```cpp
RfidCore(CardHandler* handler);
void begin();                          // Initialize RFID readers
void update();                         // Read all readers and trigger events
bool isPairComplete(int pairId);       // Check if pair has all cards
std::vector<int> listPairID();         // Get list of pair IDs
```

#### 3. `config.h`
Shared configuration and constants

**Features:**
- I2C addresses and pin definitions
- Client type constants
- Helper macros for build flags
- Reader count calculation

## Usage

### In WiFi Client (`m5stack`)

```cpp
#include <rfid_core.h>
#include <config.h>
#include "http_card_handler.h"

HttpCardHandler cardHandler(deviceId, apiHost);
RfidCore rfidCore(&cardHandler);

void setup() {
  cardHandler.onBoot("power_on");
  rfidCore.begin();
}

void loop() {
  rfidCore.update();
  if (rfidCore.isPairComplete(1)) {
    // Both cards detected
  }
}
```

### In Wired Client (`m5stack-wired`)

```cpp
#include <rfid_core.h>
#include <config.h>
#include "serial_card_handler.h"

SerialCardHandler cardHandler;
RfidCore rfidCore(&cardHandler);

void setup() {
  Serial.begin(115200);
  cardHandler.onBoot("power_on");
  rfidCore.begin();
}

void loop() {
  rfidCore.update();
}
```

## Client Types

Defined in `config.h`:

- **`player`**: 2 RFID readers for hole cards, sends only when both detected
- **`board`**: 5 RFID readers for community cards, sends each immediately
- **`muck`**: Same as player mode, for muck pile cards

Set via build flag:
```bash
-DCLIENT_TYPE=player
```

## Hardware Support

- **RFID Module**: RC522 via I2C (address 0x28)
- **I2C Multiplexer**: TCA9548A (address 0x70)
- **Reset Pin**: GPIO 12
- **I2C Clock**: 100kHz

## Code Reduction

By using this library:

- **~200-250 lines** of duplicate code eliminated
- **40%** reduction in total codebase
- Single point of maintenance for RFID logic
- Consistent behavior across all clients

## File Structure

```
lib/rfid-poker-common/
├── library.properties     # PlatformIO library metadata
├── README.md             # This file
└── src/
    ├── card_handler.h    # Abstract interface for card events
    ├── config.h          # Shared configuration
    ├── rfid_core.h       # Core RFID logic header
    └── rfid_core.cpp     # Core RFID logic implementation
```

## Dependencies

- `MFRC522_I2C` (v1.0+)
- `ClosedCube TCA9548A` (v2020.5.21+)
- `ClosedCube I2C Driver` (v2020.9.8+)

## Integration

Add to your `platformio.ini`:

```ini
lib_extra_dirs = ../../lib
```

The library will be automatically discovered and linked.

## Extending

To add a new client type:

1. Create a new `CardHandler` implementation
2. Implement `onCardDetected()`, `onError()`, and `onBoot()`
3. Create `RfidCore` instance with your handler
4. Call `begin()` in setup and `update()` in loop

Example for Bluetooth client:

```cpp
class BluetoothCardHandler : public CardHandler {
  void onCardDetected(int channel, const char* uid) override {
    // Send via Bluetooth
  }
  void onError(const char* code, const char* message) override {
    // Log error via Bluetooth
  }
  void onBoot(const char* reason) override {
    // Announce boot via Bluetooth
  }
};
```

## Testing

Each client can be tested independently. The library has no external dependencies beyond the RFID hardware libraries.

## License

Same as parent project.
