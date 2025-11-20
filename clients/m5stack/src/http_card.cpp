#include <Arduino.h>
#include <ArduinoJson.h>
#include <HTTPClient.h>

HTTPClient http;

void postCard(String macAddr, String uid, int pair_id, String i_host) {
  unsigned long startTime = millis();
  Serial.printf("\n[POST START] pair_id=%d, uid=%s, time=%lu ms\n", pair_id,
                uid.c_str(), startTime);
  Serial.flush();

  StaticJsonDocument<256> json_request;
  char buffer[255];

  json_request["device_id"] = macAddr;
  json_request["uid"] = uid;
  json_request["pair_id"] = pair_id;

  // Verify JSON size before serialization
  size_t jsonSize = measureJson(json_request);
  if (jsonSize >= sizeof(buffer)) {
    Serial.printf("Error: JSON payload too large\n");
    Serial.flush();
    return;
  }

  serializeJson(json_request, buffer);
  Serial.printf("%s\n", buffer);
  Serial.flush();

  unsigned long beforeConnect = millis();
  Serial.printf("[HTTP] Connecting to %s (elapsed: %lu ms)\n",
                (i_host + "/card").c_str(), beforeConnect - startTime);
  Serial.flush();

  http.setTimeout(10000); // 10 seconds
  // http.setReuse(false);  // Disable connection reuse to avoid keep-alive issues
  http.begin(i_host + "/card");
  http.addHeader("Content-Type", "application/json");

  unsigned long beforePost = millis();
  Serial.printf("[HTTP] Connection established (elapsed: %lu ms)\n",
                beforePost - startTime);
  Serial.flush();

  int maxRetries = 3; // 3 attempts with 5-second timeout each
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
      delay(100); // Wait 100ms before retry
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
  Serial.printf("[POST END] pair_id=%d, total time=%lu ms\n\n", pair_id,
                endTime - startTime);
  Serial.flush();
};
