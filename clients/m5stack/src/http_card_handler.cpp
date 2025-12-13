#include "http_card_handler.h"
#include <ArduinoJson.h>
#include <HTTPClient.h>
#include <config.h>

HttpCardHandler::HttpCardHandler(const String& deviceId, const String& apiHost)
    : _deviceId(deviceId), _apiHost(apiHost) {}

int HttpCardHandler::getPairID(int channelId) {
  const char* clientType = getClientType();

  if (strcmp(clientType, ClientType::PLAYER) == 0 ||
      strcmp(clientType, ClientType::MUCK) == 0) {
    return 1;
  } else if (strcmp(clientType, ClientType::BOARD) == 0) {
    return channelId + 1;
  }

  switch (channelId) {
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

void HttpCardHandler::onCardDetected(int channel, const char* uid) {
  int pairId = getPairID(channel);
  postCard(String(uid), pairId);
}

void HttpCardHandler::postCard(const String& uid, int pairId) {
  unsigned long startTime = millis();
  Serial.printf("\n[POST START] pair_id=%d, uid=%s, time=%lu ms\n", pairId,
                uid.c_str(), startTime);
  Serial.flush();

  StaticJsonDocument<256> json_request;
  char buffer[255];

  json_request["device_id"] = _deviceId;
  json_request["uid"] = uid;
  json_request["pair_id"] = pairId;

  size_t jsonSize = measureJson(json_request);
  if (jsonSize >= sizeof(buffer)) {
    Serial.printf("Error: JSON payload too large\n");
    Serial.flush();
    return;
  }

  serializeJson(json_request, buffer);
  Serial.printf("%s\n", buffer);
  Serial.flush();

  HTTPClient http;
  unsigned long beforeConnect = millis();
  Serial.printf("[HTTP] Connecting to %s (elapsed: %lu ms)\n",
                (_apiHost + "/card").c_str(), beforeConnect - startTime);
  Serial.flush();

  http.setTimeout(10000);
  http.begin(_apiHost + "/card");
  http.addHeader("Content-Type", "application/json");

  unsigned long beforePost = millis();
  Serial.printf("[HTTP] Connection established (elapsed: %lu ms)\n",
                beforePost - startTime);
  Serial.flush();

  int maxRetries = 3;
  int httpCode;
  for (int i = 0; i < maxRetries; i++) {
    unsigned long retryStart = millis();
    Serial.printf("[HTTP] POST attempt %d/%d (elapsed: %lu ms)\n", i + 1,
                  maxRetries, retryStart - startTime);
    Serial.flush();

    httpCode = http.POST(buffer);

    unsigned long retryEnd = millis();
    Serial.printf("[HTTP] POST attempt %d result: %d (took %lu ms, total "
                  "elapsed: %lu ms)\n",
                  i + 1, httpCode, retryEnd - retryStart, retryEnd - startTime);
    Serial.flush();

    if (httpCode > 0)
      break;

    if (i < maxRetries - 1) {
      Serial.printf("[HTTP] Retrying after 100ms...\n");
      Serial.flush();
      delay(100);
    }
  }

  if (httpCode > 0) {
    unsigned long beforeGetString = millis();
    Serial.printf("[HTTP] Getting response payload (elapsed: %lu ms)\n",
                  beforeGetString - startTime);
    Serial.flush();

    String payload = http.getString();

    unsigned long afterGetString = millis();
    Serial.printf("[HTTP] Response code: %d (payload retrieval took %lu ms)\n",
                  httpCode, afterGetString - beforeGetString);
    Serial.printf("%s\n", payload.c_str());
    Serial.flush();
  } else {
    Serial.printf("Error on sending POST: %s\n",
                  http.errorToString(httpCode).c_str());
    Serial.flush();
  }

  http.end();

  unsigned long endTime = millis();
  Serial.printf("[POST END] pair_id=%d, total time=%lu ms\n\n", pairId,
                endTime - startTime);
  Serial.flush();
}

void HttpCardHandler::onError(const char* code, const char* message) {
  Serial.printf("[ERROR] Code: %s, Message: %s\n", code, message);
  Serial.flush();
}

void HttpCardHandler::onBoot(const char* reason) {
  // Get pair IDs based on client configuration
  std::vector<int> antenna_ids;
  int readerCount = getRfidReaderCount();
  if (readerCount <= 2) {
    antenna_ids = {1};
  } else if (readerCount <= 4) {
    antenna_ids = {1, 2};
  } else if (readerCount <= 6) {
    antenna_ids = {1, 2, 3};
  }

  StaticJsonDocument<256> json_request;
  char buffer[255];

  json_request["device_id"] = _deviceId;
  JsonArray value = json_request["pair_ids"].to<JsonArray>();
  for (int i = 0; i < antenna_ids.size(); i++) {
    value.add(antenna_ids[i]);
  }

  size_t jsonSize = measureJson(json_request);
  if (jsonSize >= sizeof(buffer)) {
    Serial.println("Error: JSON payload too large");
    return;
  }

  serializeJson(json_request, buffer);
  Serial.println(buffer);

  HTTPClient http;
  http.setTimeout(5000);
  http.begin(_apiHost + "/device/boot");
  http.addHeader("Content-Type", "application/json");

  int maxRetries = 3;
  int httpCode;
  for (int i = 0; i < maxRetries; i++) {
    httpCode = http.POST(buffer);
    if (httpCode > 0)
      break;
    delay(1000);
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
}
