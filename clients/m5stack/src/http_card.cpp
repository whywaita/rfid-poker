#include <HTTPClient.h>
#include <ArduinoJson.h>

void postCard(String macAddr, String uid, int pair_id, String i_host) {
    StaticJsonDocument<256> json_request;
    char buffer[255];

    json_request["device_id"] = macAddr;
    json_request["uid"] = uid;
    json_request["pair_id"] = pair_id;

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
    http.begin(i_host+"/card");
    http.addHeader("Content-Type", "application/json");

    int maxRetries = 3;
    int httpCode;
    for (int i = 0; i < maxRetries; i++) {
        httpCode = http.POST(buffer);
        if (httpCode > 0) break;
        delay(1000); // Wait 1 second before retry
    }

    if (httpCode > 0) {
        String payload = http.getString();
        Serial.println(httpCode);
        Serial.println(payload);
    } else {
        Serial.println("Error on sending POST: " + http.errorToString(httpCode));
    }
    http.end();
};