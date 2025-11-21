#include "serial_card_handler.h"
#include <config.h>

const char* SerialCardHandler::getDeviceID() {
  // Use DEVICE_ID if specified and not empty, otherwise use MAC address
#ifdef DEVICE_ID
  const char* deviceId = TOSTRING(DEVICE_ID);
  if (deviceId != nullptr && deviceId[0] != '\0') {
    return deviceId;
  }
#endif
  // Return MAC address as device ID (set during initialization)
  static char macStr[18] = {0};
  if (macStr[0] == 0) {
    // Get MAC address on first call
    uint8_t mac[6];
    esp_read_mac(mac, ESP_MAC_WIFI_STA);
    snprintf(macStr, sizeof(macStr), "%02X:%02X:%02X:%02X:%02X:%02X",
             mac[0], mac[1], mac[2], mac[3], mac[4], mac[5]);
  }
  return macStr;
}

const char* SerialCardHandler::getFirmwareVersion() {
#ifdef FW_VERSION
  const char* fwVersion = TOSTRING(FW_VERSION);
  if (fwVersion != nullptr && fwVersion[0] != '\0') {
    return fwVersion;
  }
#endif
  return "unknown";
}

String SerialCardHandler::getTimestamp() {
  unsigned long ms = millis();
  char buffer[32];
  // Simple format: milliseconds since boot
  snprintf(buffer, sizeof(buffer), "T+%lu", ms);
  return String(buffer);
}

unsigned long SerialCardHandler::getNextSequence() {
  return ++_sequenceCounter;
}

void SerialCardHandler::onCardDetected(int channel, const char* uid) {
  JsonDocument json;

  json["type"] = "card";
  json["ts"] = getTimestamp();
  json["device_id"] = getDeviceID();
  json["seq"] = getNextSequence();
  json["card_uid"] = uid;
  json["tech"] = "MIFARE";
  json["rssi"] = 0;

  serializeJson(json, Serial);
  Serial.println(); // End with newline for JSON Lines format
  Serial.flush();
}

void SerialCardHandler::onError(const char* code, const char* message) {
  JsonDocument json;

  json["type"] = "error";
  json["ts"] = getTimestamp();
  json["device_id"] = getDeviceID();
  json["seq"] = getNextSequence();
  json["code"] = code;
  json["message"] = message;

  serializeJson(json, Serial);
  Serial.println(); // End with newline for JSON Lines format
  Serial.flush();
}

void SerialCardHandler::onBoot(const char* reason) {
  JsonDocument json;

  json["type"] = "boot";
  json["ts"] = getTimestamp();
  json["device_id"] = getDeviceID();
  json["seq"] = getNextSequence();
  json["fw_version"] = getFirmwareVersion();
  json["reason"] = reason;

  serializeJson(json, Serial);
  Serial.println(); // End with newline for JSON Lines format
  Serial.flush();
}
