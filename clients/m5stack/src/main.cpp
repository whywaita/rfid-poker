#include <Arduino.h>
#include <FastLED.h>
#include <M5Atom.h>
#include <WiFi.h>
#include <tuple>

#include <rfid_core.h>
#include <config.h>
#include "http_card_handler.h"

std::tuple<String, String> setupNetwork();

// Global variables
char macStr[18];
String apiHost;
HttpCardHandler* cardHandler = nullptr;
RfidCore* rfidCore = nullptr;

void setupAtom() {
  M5.begin(true, false, true); // Enable Serial, disable I2C, enable display

  // Setup Mac Address
  uint8_t mac[6];
  esp_read_mac(mac, ESP_MAC_WIFI_STA);
  snprintf(macStr, sizeof(macStr), "%02X:%02X:%02X:%02X:%02X:%02X", mac[0],
           mac[1], mac[2], mac[3], mac[4], mac[5]);

  Serial.println("RFID Reader");
  Serial.printf("Mac: %s\n", macStr);
  Serial.printf("SSID: %s\n", "Not connected");

  String ssid;
  try {
    std::tie(ssid, apiHost) = setupNetwork();
  } catch (const std::exception &e) {
    Serial.println("Network Error:");
    Serial.println(e.what());
    delay(5000);
    ESP.restart();
  }

  Serial.println("RFID Reader");
  Serial.printf("Mac: %s\n", macStr);
  Serial.printf("SSID: %s\n", ssid.c_str());

  // Initialize card handler and RFID core
  cardHandler = new HttpCardHandler(String(macStr), apiHost);
  rfidCore = new RfidCore(cardHandler);

  // Send boot message
  cardHandler->onBoot("power_on");

  // Setup RFID readers
  rfidCore->begin();

  // Display CLIENT_TYPE and reader count
  const char *clientType = getClientType();
  int readerCount = getRfidReaderCount();
  Serial.printf("CLIENT_TYPE: %s\n", clientType);
  Serial.printf("RFID Reader Count: %d\n", readerCount);
}

void loopAtom() {
  if (rfidCore == nullptr) {
    Serial.println("Error: RfidCore not initialized");
    delay(1000);
    return;
  }

  // Update RFID readers
  rfidCore->update();

  // Check if pair 1 is complete (we assume ATOM has 2 antennas = 1 pair)
  if (rfidCore->isPairComplete(1)) {
    // Light up the LED with green color when both antennas detect cards
    M5.dis.drawpix(0, 0x00ff00); // Green
  } else {
    // Turn off the LED when not all antennas detect cards
    M5.dis.drawpix(0, 0x000000); // Off
  }

  M5.update();
  delay(200); // Reduced from 1000ms for faster response
}

void setup() { setupAtom(); }

void loop() { loopAtom(); }
