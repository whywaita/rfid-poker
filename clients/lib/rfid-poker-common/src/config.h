#ifndef RFID_POKER_CONFIG_H
#define RFID_POKER_CONFIG_H

// Helper macros to stringify the macro values
#define STRINGIFY(x) #x
#define TOSTRING(x) STRINGIFY(x)

// I2C Configuration
#define PaHub_I2C_ADDRESS 0x70
#define RFID_ADDRESS 0x28
#define PIN_RESET 12

// Card history cooldown
// For wired clients: 0.5 seconds (faster response for serial communication)
// For other clients: 10 seconds (avoid overwhelming network)
#ifdef WIRED_CLIENT
#define CARD_SEND_COOLDOWN_MS 500
#else
#define CARD_SEND_COOLDOWN_MS 10000
#endif

// Maximum number of RFID readers supported
#define MAX_RFID_READERS 6

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
