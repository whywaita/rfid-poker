#include <Arduino.h>
#include <WiFi.h>

#include <M5Core2.h>
#include <Wire.h>
#include <MFRC522_I2C.h>
#include "ClosedCube_TCA9548A.h"
 
#define WIRE Wire
#define PaHub_I2C_ADDRESS 0x70
#define RFID_READER_COUNT 2  // The number of RFID readers

#define RFID_ADDRESS 0x28    // The I2C address of the RFID reader
#define PIN_RESET 12
MFRC522_I2C mfrc522(RFID_ADDRESS, PIN_RESET, &Wire);

ClosedCube::Wired::TCA9548A tca;

void tcaselect(uint8_t i);
void readAllRfid();
void setupRfId();
String readUID();
bool hasCard();

void setup() {    
    M5.begin();
    M5.Lcd.fillScreen(BLACK);
    M5.lcd.setTextSize(2);

    // Setup Mac Address
    uint8_t mac[6];
    esp_read_mac(mac, ESP_MAC_WIFI_STA);
    char macStr[18];
    snprintf(macStr, sizeof(macStr), "%02X:%02X:%02X:%02X:%02X:%02X",
           mac[0], mac[1], mac[2], mac[3], mac[4], mac[5]);

    M5.Lcd.setCursor(0, 0);
    M5.Lcd.setTextSize(2);
    M5.Lcd.println("RFID Reader");
    M5.Lcd.printf("Mac: %s\n", macStr);
    M5.Lcd.printf("SSID: %s\n", "SSID");  // TODO: Configure your SSID
    // boarder line between fixed and dynamic parts
    M5.Lcd.drawLine(0, 50, M5.Lcd.width(), 50, WHITE);

    setupRfId();
}
 
void loop() {
  // Remove the dynamic display part
  M5.Lcd.fillRect(0, 51, M5.Lcd.width(), M5.Lcd.height() - 51, BLACK);
  
  // Set the cursor to the dynamic display part
  M5.Lcd.setCursor(0, 51);
  readAllRfid();
  delay(1000);
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
 
void readAllRfid() {
    for (int channel = 0; channel < RFID_READER_COUNT; channel++) {
        tcaselect(channel);
        String uid = readUID();
        if (uid == "") {
          continue;
        }
        M5.Lcd.printf("[c%d] ", channel);
        M5.Lcd.print(uid);
        M5.Lcd.println("");

        Serial.print("[Channel] ");
        Serial.print(channel);
        Serial.printf(" [UID: %s]", uid.c_str());
        Serial.println("");
    }
}

String readUID() {
  if (!hasCard()) {
    // Do nothing if there is no card
    return "";
  }

  String uid = "";
  for (byte i=0; i<mfrc522.uid.size; i++) {
    uid += mfrc522.uid.uidByte[i] < 0x10 ? " 0" : " ";
    uid += String(mfrc522.uid.uidByte[i], HEX);
  }

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