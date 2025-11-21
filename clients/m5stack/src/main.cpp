#include <Arduino.h>
#include <FastLED.h>
#include <M5Atom.h>
#include <WiFi.h>
#include <tuple>

void readAllRfid(char macAddr[], String i_host);
void setupRfId();
std::tuple<String, String> setupNetwork();
void postDeviceBoot(String macAddr, String i_host);
char macStr[18];
String i_host;

// Add these declarations
extern bool isPairComplete(int pair_id);
extern const char *getClientType();
extern int getRfidReaderCount();

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

  String i_ssid;

  try {
    std::tie(i_ssid, i_host) = setupNetwork();
  } catch (const std::exception &e) {
    Serial.println("Network Error:");
    Serial.println(e.what());
    delay(5000);
    ESP.restart();
  }

  Serial.println("RFID Reader");
  Serial.printf("Mac: %s\n", macStr);
  Serial.printf("SSID: %s\n", i_ssid);

  postDeviceBoot(macStr, i_host);

  setupRfId();

  // Display CLIENT_TYPE and reader count
  const char *clientType = getClientType();
  int readerCount = getRfidReaderCount();
  Serial.printf("CLIENT_TYPE: %s\n", clientType);
  Serial.printf("RFID Reader Count: %d\n", readerCount);
}

void loopAtom() {
  try {
    readAllRfid(macStr, i_host);

    // Check if pair 1 is complete (we assume ATOM has 2 antennas = 1 pair)
    if (isPairComplete(1)) {
      // Light up the LED with green color when both antennas detect cards
      M5.dis.drawpix(0, 0x00ff00); // Green
    } else {
      // Turn off the LED when not all antennas detect cards
      M5.dis.drawpix(0, 0x000000); // Off
    }

  } catch (const std::exception &e) {
    Serial.println("RFID Error:");
    Serial.println(e.what());
    // Blink red LED on error
    M5.dis.drawpix(0, 0xff0000); // Red
  }

  M5.update();
  delay(200); // Reduced from 1000ms for faster response
}

void setup() { setupAtom(); }

void loop() { loopAtom(); }