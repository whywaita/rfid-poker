#include "rfid_core.h"
#include "ClosedCube_TCA9548A.h"
#include <MFRC522_I2C.h>
#include <Wire.h>

// Static instances for RFID hardware
static MFRC522_I2C mfrc522(RFID_ADDRESS, PIN_RESET, &Wire);
static ClosedCube::Wired::TCA9548A tca;

RfidCore::RfidCore(CardHandler* handler) : _handler(handler) {
  // Initialize card detection arrays
  for (int i = 0; i < MAX_RFID_READERS; i++) {
    _cardsDetected[i] = false;
    _cardHistory[i].uid = "";
    _cardHistory[i].lastSentTime = 0;
  }
}

void RfidCore::begin() {
  Wire.begin();
  Wire.setClock(100000);
  tca.address(PaHub_I2C_ADDRESS);

  for (uint8_t t = 0; t < getRfidReaderCount(); t++) {
    selectChannel(t);
    Wire.beginTransmission(RFID_ADDRESS);
    if (Wire.endTransmission() == 0) {
      mfrc522.PCD_Init(); // Init MFRC522
    } else {
      // Send error message if RFID initialization failed
      char errorMsg[128];
      snprintf(errorMsg, sizeof(errorMsg),
               "Failed to initialize RFID reader on channel %d", t);
      _handler->onError("rfid_init_failed", errorMsg);
    }
  }
  delay(500);
}

void RfidCore::selectChannel(uint8_t channel) {
  if (channel >= getRfidReaderCount())
    return;
  Wire.beginTransmission(PaHub_I2C_ADDRESS);
  Wire.write(1 << channel); // Switch the RFID reader to be referenced by mfrc522
  Wire.endTransmission();
}

String RfidCore::readUID() {
  if (!hasCard()) {
    return "";
  }

  String val = "";
  for (byte i = 0; i < mfrc522.uid.size; i++) {
    val += mfrc522.uid.uidByte[i] < 0x10 ? " 0" : " ";
    val += String(mfrc522.uid.uidByte[i], HEX);
  }

  return val.substring(1);
}

bool RfidCore::hasCard() {
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

void RfidCore::triggerReadUID(int channel, const String& uid) {
  // Check if this card was sent recently (within cooldown period)
  unsigned long currentTime = millis();

  if (_cardHistory[channel].lastSentTime > 0) {
    unsigned long timeSinceLastSend;
    if (currentTime >= _cardHistory[channel].lastSentTime) {
      timeSinceLastSend = currentTime - _cardHistory[channel].lastSentTime;
    } else {
      // millis() has overflowed, calculate the time difference
      timeSinceLastSend =
          (0xFFFFFFFF - _cardHistory[channel].lastSentTime) + currentTime + 1;
    }

    // Check if same card was sent within cooldown period
    if (_cardHistory[channel].uid == uid &&
        timeSinceLastSend < CARD_SEND_COOLDOWN_MS) {
      // Skip sending this card
      return;
    }
  }

  // Convert String to char array
  char uidBuffer[64];
  uid.toCharArray(uidBuffer, sizeof(uidBuffer));

  // Send card detection event via handler
  _handler->onCardDetected(channel, uidBuffer);

  // Update card history after successful send
  _cardHistory[channel].uid = uid;
  _cardHistory[channel].lastSentTime = millis();
}

void RfidCore::update() {
  const char* clientType = getClientType();

  // Reset card detection status
  for (int i = 0; i < getRfidReaderCount(); i++) {
    _cardsDetected[i] = false;
  }

  // Store UIDs temporarily for player mode
  String uids[MAX_RFID_READERS] = {""};

  // Scan all channels and detect cards
  for (int channel = 0; channel < getRfidReaderCount(); channel++) {
    selectChannel(channel);
    String uid = readUID();
    if (uid != "") {
      _cardsDetected[channel] = true;
      uids[channel] = uid;
    }
  }

  if (strcmp(clientType, ClientType::PLAYER) == 0 ||
      strcmp(clientType, ClientType::MUCK) == 0) {
    // Player/Muck mode: send only when both cards are detected
    if (_cardsDetected[0] && _cardsDetected[1]) {
      triggerReadUID(0, uids[0]);
      triggerReadUID(1, uids[1]);
    }
  } else if (strcmp(clientType, ClientType::BOARD) == 0) {
    // Board mode: send each card immediately with small delay between requests
    for (int channel = 0; channel < getRfidReaderCount(); channel++) {
      if (_cardsDetected[channel]) {
        triggerReadUID(channel, uids[channel]);
        // Small delay to avoid overwhelming the output
        if (channel < getRfidReaderCount() - 1) {
          delay(100); // 100ms delay between messages
        }
      }
    }
  } else {
    // Fallback: original behavior (send immediately)
    for (int channel = 0; channel < getRfidReaderCount(); channel++) {
      if (_cardsDetected[channel]) {
        triggerReadUID(channel, uids[channel]);
      }
    }
  }
}

bool RfidCore::isPairComplete(int pairId) const {
  switch (pairId) {
  case 1:
    return _cardsDetected[0] && _cardsDetected[1];
  case 2:
    return _cardsDetected[2] && _cardsDetected[3];
  case 3:
    return _cardsDetected[4] && _cardsDetected[5];
  default:
    return false;
  }
}

int RfidCore::getPairID(int channelId) const {
  const char* clientType = getClientType();

  if (strcmp(clientType, ClientType::PLAYER) == 0 ||
      strcmp(clientType, ClientType::MUCK) == 0) {
    // Player/Muck mode: both channels (0 and 1) belong to pair 1
    return 1;
  } else if (strcmp(clientType, ClientType::BOARD) == 0) {
    // Board mode: each channel is independent
    return channelId + 1; // channels 0-4 map to pair_ids 1-5
  }

  // Fallback to original behavior (2 channels per pair)
  switch (channelId) {
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

std::vector<int> RfidCore::listPairID() const {
  if (getRfidReaderCount() <= 2) {
    return {1};
  } else if (getRfidReaderCount() <= 4) {
    return {1, 2};
  } else if (getRfidReaderCount() <= 6) {
    return {1, 2, 3};
  }
  return {};
}
