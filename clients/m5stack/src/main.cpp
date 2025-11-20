#include <Arduino.h>
#include <WiFi.h>
#include <tuple>

#ifdef M5STACK_CORE2
#include <M5Unified.h>
#elif defined(M5STACK_ATOM)
#include <FastLED.h>
#include <M5Atom.h>
#endif

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

#ifdef M5STACK_CORE2
void setupCore2() {
  M5.begin();
  M5.Lcd.fillScreen(BLACK);
  M5.Lcd.setTextSize(2);

  // Setup Mac Address
  uint8_t mac[6];
  esp_read_mac(mac, ESP_MAC_WIFI_STA);
  snprintf(macStr, sizeof(macStr), "%02X:%02X:%02X:%02X:%02X:%02X", mac[0],
           mac[1], mac[2], mac[3], mac[4], mac[5]);

  M5.Lcd.setCursor(0, 0);
  M5.Lcd.println("RFID Reader");
  M5.Lcd.printf("Mac: %s\n", macStr);
  M5.Lcd.printf("SSID: %s\n", "Not connected");
  // boarder line between fixed and dynamic parts
  M5.Lcd.drawLine(0, 50, M5.Lcd.width(), 50, WHITE);

  String i_ssid;

  try {
    std::tie(i_ssid, i_host) = setupNetwork();
  } catch (const std::exception &e) {
    M5.Lcd.println("Network Error:");
    M5.Lcd.println(e.what());
    delay(5000);
    ESP.restart();
  }

  M5.Lcd.fillScreen(BLACK);
  M5.Lcd.setCursor(0, 0);
  M5.Lcd.println("RFID Reader");
  M5.Lcd.printf("Mac: %s\n", macStr);
  M5.Lcd.printf("SSID: %s\n", i_ssid);
  // boarder line between fixed and dynamic parts
  M5.Lcd.drawLine(0, 50, M5.Lcd.width(), 50, WHITE);

  postDeviceBoot(macStr, i_host);

  setupRfId();

  // Display CLIENT_TYPE and reader count
  const char *clientType = getClientType();
  int readerCount = getRfidReaderCount();
  M5.Lcd.printf("Type: %s\n", clientType);
  M5.Lcd.printf("Readers: %d\n", readerCount);
  Serial.printf("CLIENT_TYPE: %s\n", clientType);
  Serial.printf("RFID Reader Count: %d\n", readerCount);
}

void loopCore2() {
  // Remove the dynamic display part
  M5.Lcd.fillRect(0, 51, M5.Lcd.width(), M5.Lcd.height() - 51, BLACK);

  // Set the cursor to the dynamic display part
  M5.Lcd.setCursor(0, 51);

  try {
    readAllRfid(macStr, i_host);
  } catch (const std::exception &e) {
    M5.Lcd.println("RFID Error:");
    M5.Lcd.println(e.what());
  }
  delay(200); // Reduced from 1000ms for faster response
}
#elif defined(M5STACK_ATOM)
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
#endif

void setup() {
#ifdef M5STACK_CORE2
  setupCore2();
#elif defined(M5STACK_ATOM)
  setupAtom();
#else
#error "Unsupported device. Please define either M5STACK_CORE2 or M5STACK_ATOM."
#endif
}

void loop() {
#ifdef M5STACK_CORE2
  loopCore2();
#elif defined(M5STACK_ATOM)
  loopAtom();
#endif
}