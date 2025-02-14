#include <vector>

#include <M5Core2.h>
#include <Wire.h>
#include <MFRC522_I2C.h>
#include "ClosedCube_TCA9548A.h"

#include <HTTPClient.h>
#include <ArduinoJson.h>

#define WIRE Wire
#define PaHub_I2C_ADDRESS 0x70
#define RFID_READER_COUNT 4  // The number of RFID readers

#define RFID_ADDRESS 0x28    // The I2C address of the RFID reader
#define PIN_RESET 12
MFRC522_I2C mfrc522(RFID_ADDRESS, PIN_RESET, &Wire);

ClosedCube::Wired::TCA9548A tca;

void tcaselect(uint8_t i);
void readAllRfid(String macAddr, String i_host);
void setupRfId();
String readUID();
bool hasCard();
int getPairID(int channel_id);

void postCardAsync(String macAddr, String uid, int pair_id, String i_host);

void triggerReadUID(int channel, String uid, String macAddr, String i_host) {
    M5.Lcd.printf("[c%d] ", channel);
    M5.Lcd.print(uid);
    M5.Lcd.println("");

    Serial.print("[Channel] ");
    Serial.print(channel);
    Serial.printf(" [UID: %s]", uid.c_str());
    Serial.println("");

    int pair_id = getPairID(channel);
    postCardAsync(macAddr, uid, pair_id, i_host);
}

void setupRfId() {
    Wire.begin();
    Wire.setClock(100000);
    tca.address(PaHub_I2C_ADDRESS);
    for (uint8_t t = 0; t < RFID_READER_COUNT; t++) {
        tcaselect(t);
        Wire.beginTransmission(RFID_ADDRESS);
        if (Wire.endTransmission() == 0) {
            mfrc522.PCD_Init();          // Init MFRC522
        }
    }
    delay(500); 
}

void tcaselect(uint8_t i) {
    if (i >= RFID_READER_COUNT) return;
    Wire.beginTransmission(PaHub_I2C_ADDRESS);
    Wire.write(1 << i); // Switch the RFID reader to be referenced by mfrc522
    Wire.endTransmission();
}
 
void readAllRfid(char macAddr[], String i_host) {
    for (int channel = 0; channel < RFID_READER_COUNT; channel++) {
        tcaselect(channel);
        String uid = readUID();
        if (uid == "") {
          continue;
        };
        triggerReadUID(channel, uid, macAddr, i_host);
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
    if (RFID_READER_COUNT <= 2) {
        return {1};
    } else if (RFID_READER_COUNT <= 4) {
        return {1, 2};
    } else if (RFID_READER_COUNT <= 6) {
        return {1, 2, 3};
    }
    return {};
}