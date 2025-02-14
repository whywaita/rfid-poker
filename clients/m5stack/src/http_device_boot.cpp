#include <HTTPClient.h>
#include <ArduinoJson.h>

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
    JsonArray value = json_request.createNestedArray("pair_ids");
    for (int i = 0; i < antenna_ids.size(); i++) {
        value.add(antenna_ids[i]);
    }

    serializeJson(json_request, buffer);
    Serial.println(buffer);

    HTTPClient http;
    http.begin(i_host+"/device/boot");
    http.addHeader("Content-Type", "application/json");
    int httpCode = http.POST(buffer);
    if (httpCode > 0) {
        String payload = http.getString();
        Serial.println(httpCode);
        Serial.println(payload);
    } else {
        Serial.println("Error on sending POST: " + http.errorToString(httpCode));
    }
    http.end();
};