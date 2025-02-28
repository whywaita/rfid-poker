#include <vector>

#ifdef M5STACK_CORE2
#include <M5Unified.h>
#endif

#include <Wire.h>
#include <MFRC522_I2C.h>
#include "ClosedCube_TCA9548A.h"

#include <HTTPClient.h>
#include <ArduinoJson.h>

#define WIRE Wire
#define PaHub_I2C_ADDRESS 0x70

#define RFID_ADDRESS 0x28    // The I2C address of the RFID reader
#define PIN_RESET 12
MFRC522_I2C mfrc522(RFID_ADDRESS, PIN_RESET, &Wire);

ClosedCube::Wired::TCA9548A tca;

// Add function to get RFID reader count based on device type
int getRfidReaderCount() {
#ifdef M5STACK_CORE2
    return 6;  // Core2 has 6 RFID readers
#elif defined(M5STACK_ATOM)
    return 2;  // Atom has 2 RFID readers
#else
    return 0;  // Unknown device
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
bool cardsDetected[6] = {false};  // Use maximum size (6) for array

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
    // Only output to LCD when using M5Stack Core2
#ifdef M5STACK_CORE2
    M5.Lcd.printf("[c%d] ", channel);
    M5.Lcd.print(uid);
    M5.Lcd.println("");
#endif

    // Always output to Serial for debugging
    Serial.print("[Channel] ");
    Serial.print(channel);
    Serial.printf(" [UID: %s]", uid.c_str());
    Serial.println("");

    int pair_id = getPairID(channel);
    postCard(macAddr, uid, pair_id, i_host);
}

void setupRfId() {
    Wire.begin();
    Wire.setClock(100000);
    tca.address(PaHub_I2C_ADDRESS);
    for (uint8_t t = 0; t < getRfidReaderCount(); t++) {
        tcaselect(t);
        Wire.beginTransmission(RFID_ADDRESS);
        if (Wire.endTransmission() == 0) {
            mfrc522.PCD_Init();          // Init MFRC522
        }
    }
    delay(500); 
}

void tcaselect(uint8_t i) {
    if (i >= getRfidReaderCount()) return;
    Wire.beginTransmission(PaHub_I2C_ADDRESS);
    Wire.write(1 << i); // Switch the RFID reader to be referenced by mfrc522
    Wire.endTransmission();
}
 
void readAllRfid(char macAddr[], String i_host) {
    // Reset card detection status
    for (int i = 0; i < getRfidReaderCount(); i++) {
        cardsDetected[i] = false;
    }
    
    for (int channel = 0; channel < getRfidReaderCount(); channel++) {
        tcaselect(channel);
        String uid = readUID();
        if (uid != "") {
            cardsDetected[channel] = true;
            triggerReadUID(channel, uid, macAddr, i_host);
        }
    }
    
    // Check if any pair is complete (for debugging)
    for (int pair_id : listPairID()) {
        if (isPairComplete(pair_id)) {
            Serial.printf("Pair %d is complete!\n", pair_id);
        }
    }
}

String readUID() {
  if (!hasCard()) {
    // Do nothing if there is no card
    return "";
  }

  String val = "";
  for (byte i=0; i<mfrc522.uid.size; i++) {
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