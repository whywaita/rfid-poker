#include <vector>

#ifdef M5STACK_CORE2
#include <M5Unified.h>
#endif

#include "ClosedCube_TCA9548A.h"
#include <MFRC522_I2C.h>
#include <Wire.h>

#include <ArduinoJson.h>
#include <HTTPClient.h>

#define WIRE Wire
#define PaHub_I2C_ADDRESS 0x70

#define RFID_ADDRESS 0x28 // The I2C address of the RFID reader
#define PIN_RESET 12
MFRC522_I2C mfrc522(RFID_ADDRESS, PIN_RESET, &Wire);

ClosedCube::Wired::TCA9548A tca;

// Helper macros to stringify the CLIENT_TYPE macro
#define STRINGIFY(x) #x
#define TOSTRING(x) STRINGIFY(x)

// Get client type from environment variable
const char *getClientType() {
#ifdef CLIENT_TYPE
  return TOSTRING(CLIENT_TYPE);
#else
  return "unknown";
#endif
}

// Add function to get RFID reader count based on client type
int getRfidReaderCount() {
  const char *clientType = getClientType();

  if (strcmp(clientType, "player") == 0 || strcmp(clientType, "muck") == 0) {
    return 2; // Player/Muck mode: 2 RFID readers for 2 hole cards
  } else if (strcmp(clientType, "board") == 0) {
    return 5; // Board mode: 5 RFID readers for community cards
  }

  // Fallback to device type if CLIENT_TYPE not specified
#ifdef M5STACK_CORE2
  return 6; // Core2 has 6 RFID readers
#elif defined(M5STACK_ATOM)
  return 2; // Atom has 2 RFID readers
#else
  return 0; // Unknown device
#endif
}

void tcaselect(uint8_t i);
void readAllRfid(char macAddr[], String i_host);
void setupRfId();
String readUID();
bool hasCard();
int getPairID(int channel_id);
std::vector<int> listPairID();
// Track cards detected on each channel - use maximum possible size
bool cardsDetected[6] = {false}; // Use maximum size (6) for array

// Card history to prevent duplicate sends within 30 seconds
#define CARD_SEND_COOLDOWN_MS 30000 // 30 seconds in milliseconds

struct CardHistory {
  String uid;
  unsigned long lastSentTime;
};

// Store card history for each channel (max 6 channels)
CardHistory cardHistory[6] = {{"", 0}};

// Check if both antennas in a pair have cards
bool isPairComplete(int pair_id) {
  switch (pair_id) {
  case 1:
    return cardsDetected[0] && cardsDetected[1];
  case 2:
    return cardsDetected[2] && cardsDetected[3];
  case 3:
    return cardsDetected[4] && cardsDetected[5];
  default:
    return false;
  }
}

void postCard(String macAddr, String uid, int pair_id, String i_host);

void triggerReadUID(int channel, String uid, char macAddr[], String i_host) {
  // Check if this card was sent recently (within 30 seconds)
  unsigned long currentTime = millis();

  // Handle millis() overflow (occurs approximately every 50 days)
  bool timeValid = true;
  if (cardHistory[channel].lastSentTime > 0) {
    unsigned long timeSinceLastSend;
    if (currentTime >= cardHistory[channel].lastSentTime) {
      timeSinceLastSend = currentTime - cardHistory[channel].lastSentTime;
    } else {
      // millis() has overflowed, calculate the time difference
      timeSinceLastSend =
          (0xFFFFFFFF - cardHistory[channel].lastSentTime) + currentTime + 1;
    }

    // Check if same card was sent within cooldown period
    if (cardHistory[channel].uid == uid &&
        timeSinceLastSend < CARD_SEND_COOLDOWN_MS) {
      Serial.printf("\n[Channel %d] Card %s already sent %lu ms ago, skipping "
                    "(cooldown: %d ms)\n",
                    channel, uid.c_str(), timeSinceLastSend,
                    CARD_SEND_COOLDOWN_MS);
      Serial.flush();
      return; // Skip sending this card
    }
  }

  // Always output to Serial for debugging (use single printf to avoid buffer
  // issues)
  Serial.printf("\n[Channel] %d [UID: %s]\n", channel, uid.c_str());
  Serial.flush();

  // Only output to LCD when using M5Stack Core2 (after Serial to avoid
  // interference)
#ifdef M5STACK_CORE2
  M5.Lcd.printf("[c%d] ", channel);
  M5.Lcd.print(uid);
  M5.Lcd.println("");
#endif

  int pair_id = getPairID(channel);
  postCard(macAddr, uid, pair_id, i_host);

  // Update card history after successful send
  cardHistory[channel].uid = uid;
  cardHistory[channel].lastSentTime = millis();
}

void setupRfId() {
  Wire.begin();
  Wire.setClock(100000);
  tca.address(PaHub_I2C_ADDRESS);
  for (uint8_t t = 0; t < getRfidReaderCount(); t++) {
    tcaselect(t);
    Wire.beginTransmission(RFID_ADDRESS);
    if (Wire.endTransmission() == 0) {
      mfrc522.PCD_Init(); // Init MFRC522
    }
  }
  delay(500);
}

void tcaselect(uint8_t i) {
  if (i >= getRfidReaderCount())
    return;
  Wire.beginTransmission(PaHub_I2C_ADDRESS);
  Wire.write(1 << i); // Switch the RFID reader to be referenced by mfrc522
  Wire.endTransmission();
}

void readAllRfid(char macAddr[], String i_host) {
  const char *clientType = getClientType();

  // Reset card detection status
  for (int i = 0; i < getRfidReaderCount(); i++) {
    cardsDetected[i] = false;
  }

  // Store UIDs temporarily for player mode
  String uids[6] = {""}; // Use maximum size

  // Scan all channels and detect cards
  for (int channel = 0; channel < getRfidReaderCount(); channel++) {
    tcaselect(channel);
    String uid = readUID();
    if (uid != "") {
      cardsDetected[channel] = true;
      uids[channel] = uid;
    }
  }

  if (strcmp(clientType, "player") == 0 || strcmp(clientType, "muck") == 0) {
    // Player/Muck mode: send only when both cards are detected
    if (cardsDetected[0] && cardsDetected[1]) {
      Serial.printf("\n%s mode: Both cards detected, sending...\n", clientType);
      Serial.flush();
      triggerReadUID(0, uids[0], macAddr, i_host);
      triggerReadUID(1, uids[1], macAddr, i_host);
    }
  } else if (strcmp(clientType, "board") == 0) {
    // Board mode: send each card immediately with small delay between requests
    for (int channel = 0; channel < getRfidReaderCount(); channel++) {
      if (cardsDetected[channel]) {
        triggerReadUID(channel, uids[channel], macAddr, i_host);
        // Small delay to avoid overwhelming the server with concurrent requests
        if (channel < getRfidReaderCount() - 1) {
          delay(100); // 100ms delay between requests
        }
      }
    }
  } else {
    // Fallback: original behavior (send immediately)
    for (int channel = 0; channel < getRfidReaderCount(); channel++) {
      if (cardsDetected[channel]) {
        triggerReadUID(channel, uids[channel], macAddr, i_host);
      }
    }
  }

  // Check if any pair is complete (for debugging)
  for (int pair_id : listPairID()) {
    if (isPairComplete(pair_id)) {
      Serial.printf("\nPair %d is complete!\n", pair_id);
      Serial.flush();
    }
  }
}

String readUID() {
  if (!hasCard()) {
    // Do nothing if there is no card
    return "";
  }

  String val = "";
  for (byte i = 0; i < mfrc522.uid.size; i++) {
    val += mfrc522.uid.uidByte[i] < 0x10 ? " 0" : " ";
    val += String(mfrc522.uid.uidByte[i], HEX);
  }

  String uid = val.substring(1);

  return uid;
}

bool hasCard() {
  if (mfrc522.PICC_IsNewCardPresent()) {
    if (mfrc522.PICC_ReadCardSerial()) {
      return true;
    }
  } else {
    if (mfrc522.PICC_IsNewCardPresent() && mfrc522.PICC_ReadCardSerial()) {
      return true;
    }
  }
  return false;
}

int getPairID(int channel_id) {
  const char *clientType = getClientType();

  if (strcmp(clientType, "player") == 0 || strcmp(clientType, "muck") == 0) {
    // Player/Muck mode: both channels (0 and 1) belong to pair 1
    return 1;
  } else if (strcmp(clientType, "board") == 0) {
    // Board mode: each channel is independent
    return channel_id + 1; // channels 0-4 map to pair_ids 1-5
  }

  // Fallback to original behavior (2 channels per pair)
  switch (channel_id) {
  case 0:
  case 1:
    return 1;
  case 2:
  case 3:
    return 2;
  case 4:
  case 5:
    return 3;
  default:
    return 0;
  }
}

std::vector<int> listPairID() {
  if (getRfidReaderCount() <= 2) {
    return {1};
  } else if (getRfidReaderCount() <= 4) {
    return {1, 2};
  } else if (getRfidReaderCount() <= 6) {
    return {1, 2, 3};
  }
  return {};
}