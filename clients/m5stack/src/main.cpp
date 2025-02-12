#include <Arduino.h>
#include <WiFi.h>

#include <M5Core2.h>

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