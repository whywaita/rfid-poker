#include <Arduino.h>
#include <WiFi.h>
#include <tuple>

#include <M5Unified.h>

void readAllRfid(char macAddr[], String i_host);
void setupRfId();

std::tuple<String, String> setupNetwork();
void postDeviceBoot(String macAddr, String i_host);

char macStr[18];
String i_host;

void setup() {    
    M5.begin();
    M5.Lcd.fillScreen(BLACK);
    M5.Lcd.setTextSize(2);

    // Setup Mac Address
    uint8_t mac[6];
    esp_read_mac(mac, ESP_MAC_WIFI_STA);
    snprintf(macStr, sizeof(macStr), "%02X:%02X:%02X:%02X:%02X:%02X",
           mac[0], mac[1], mac[2], mac[3], mac[4], mac[5]);

    M5.Lcd.setCursor(0, 0);
    M5.Lcd.println("RFID Reader");
    M5.Lcd.printf("Mac: %s\n", macStr);
    M5.Lcd.printf("SSID: %s\n", "Not connected");
    // boarder line between fixed and dynamic parts
    M5.Lcd.drawLine(0, 50, M5.Lcd.width(), 50, WHITE);

    String i_ssid;

    try {
        std::tie(i_ssid, i_host) = setupNetwork();
    } catch (const std::exception& e) {
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
}
 
void loop() {
  // Remove the dynamic display part
  M5.Lcd.fillRect(0, 51, M5.Lcd.width(), M5.Lcd.height() - 51, BLACK);
  
  // Set the cursor to the dynamic display part
  M5.Lcd.setCursor(0, 51);

  try {
    readAllRfid(macStr, i_host);
  } catch (const std::exception& e) {
    M5.Lcd.println("RFID Error:");
    M5.Lcd.println(e.what());
  }
  delay(1000);
}