#ifndef RFID_POKER_CONFIG_H
#define RFID_POKER_CONFIG_H

// Helper macros to stringify the macro values
#define STRINGIFY(x) #x
#define TOSTRING(x) STRINGIFY(x)

// I2C Configuration
#define PaHub_I2C_ADDRESS 0x70
#define RFID_ADDRESS 0x28
#define PIN_RESET 12

// Maximum number of RFID readers supported
#define MAX_RFID_READERS 6

// =============================================================================
// Timing Configuration (WIRED_CLIENT vs HTTP mode)
// =============================================================================

#ifdef WIRED_CLIENT
// --- Wired (Serial) Client Settings ---
// Faster settings for serial communication with lower latency

// Card history cooldown: time before resending the same card (ms)
#define CARD_SEND_COOLDOWN_MS 500

// Main loop delay: time between each scan cycle (ms)
#define MAIN_LOOP_DELAY_MS 50

// Board mode inter-card delay: time between sending each card in board mode (ms)
#define BOARD_INTER_CARD_DELAY_MS 10

// I2C clock speed (Hz): 400000 for fast mode, 100000 for standard mode
#define I2C_CLOCK_SPEED 400000

#else
// --- HTTP (WiFi) Client Settings ---
// Conservative settings to avoid overwhelming the network

// Card history cooldown: time before resending the same card (ms)
#define CARD_SEND_COOLDOWN_MS 10000

// Main loop delay: time between each scan cycle (ms)
#define MAIN_LOOP_DELAY_MS 200

// Board mode inter-card delay: time between sending each card in board mode (ms)
#define BOARD_INTER_CARD_DELAY_MS 100

// I2C clock speed (Hz): 100000 for standard mode (more stable over longer wires)
#define I2C_CLOCK_SPEED 100000

#endif

// Client types
namespace ClientType {
  constexpr const char* PLAYER = "player";
  constexpr const char* BOARD = "board";
  constexpr const char* MUCK = "muck";
  constexpr const char* UNKNOWN = "unknown";
}

// Get client type from build flag
inline const char* getClientType() {
#ifdef CLIENT_TYPE
  return TOSTRING(CLIENT_TYPE);
#else
  return ClientType::UNKNOWN;
#endif
}

// Get RFID reader count based on client type
inline int getRfidReaderCount() {
  const char* clientType = getClientType();

  if (strcmp(clientType, ClientType::PLAYER) == 0 ||
      strcmp(clientType, ClientType::MUCK) == 0) {
    return 2; // Player/Muck mode: 2 RFID readers for 2 hole cards
  } else if (strcmp(clientType, ClientType::BOARD) == 0) {
    return 5; // Board mode: 5 RFID readers for community cards
  }

  // Fallback to Atom if CLIENT_TYPE not specified
  return 2; // Atom has 2 RFID readers
}

#endif // RFID_POKER_CONFIG_H
