#include <HTTPClient.h>
#include <ArduinoJson.h>

struct PostCardParams {
    String macAddr;
    String uid;
    int pair_id;
    String i_host;
};

void postCardTask(void *pvParameters);
void postCard(String macAddr, String uid, int pair_id, String i_host);

void postCardAsync(String macAddr, String uid, int pair_id, String i_host) {
    PostCardParams *params = new PostCardParams{macAddr, uid, pair_id, i_host};
    
    xTaskCreate(
      postCardTask,
      "PostCardTask",
      8192,
      params,
      1,
      NULL
    );
  }

void postCardTask(void *pvParameters) {
    PostCardParams *params = (PostCardParams *)pvParameters;
    postCard(params->macAddr, params->uid, params->pair_id, params->i_host);
    delete params;
    vTaskDelete(NULL);
}

void postCard(String macAddr, String uid, int pair_id, String i_host) {
    StaticJsonDocument<256> json_request;
    char buffer[255];

    json_request["device_id"] = macAddr;
    json_request["uid"] = uid;
    json_request["pair_id"] = pair_id;

    serializeJson(json_request, buffer);
    Serial.println(buffer);

    HTTPClient http;
    http.begin(i_host+"/card");
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