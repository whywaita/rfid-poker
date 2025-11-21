#include <ArduinoJson.h>
#include <HTTPClient.h>

std::vector<int> listPairID();

struct PostDeviceParams {
  String device_id;
  std::vector<int> pair_ids;
};

void postDeviceBoot(String macAddr, String i_host) {
  std::vector<int> antenna_ids = listPairID();

  StaticJsonDocument<256> json_request;
  char buffer[255];

  json_request["device_id"] = macAddr;
  JsonArray value = json_request["pair_ids"].to<JsonArray>();
  for (int i = 0; i < antenna_ids.size(); i++) {
    value.add(antenna_ids[i]);
  }

  // Verify JSON size before serialization
  size_t jsonSize = measureJson(json_request);
  if (jsonSize >= sizeof(buffer)) {
    Serial.println("Error: JSON payload too large");
    return;
  }

  serializeJson(json_request, buffer);
  Serial.println(buffer);

  HTTPClient http;
  http.setTimeout(5000); // 5 seconds
  http.begin(i_host + "/device/boot");
  http.addHeader("Content-Type", "application/json");

  int maxRetries = 3;
  int httpCode;
  for (int i = 0; i < maxRetries; i++) {
    httpCode = http.POST(buffer);
    if (httpCode > 0)
      break;
    delay(1000); // Wait 1 second before retry
  }

  if (httpCode > 0) {
    String payload = http.getString();
    Serial.println(httpCode);
    Serial.println(payload);
  } else {
    Serial.println("Error on sending POST: " + http.errorToString(httpCode) +
                   " " + http.getString());
  }
  http.end();

  antenna_ids.clear();
};