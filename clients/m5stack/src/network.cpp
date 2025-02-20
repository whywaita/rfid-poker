#include <ArduinoJson.h>
#include <WiFi.h>
#include "SD.h"
#include <M5Unified.h>

String JsonData;
int sdstat = 0;
String i_ssid;

std::tuple<String, String> setupNetwork() {
    // Set the cursor to the dynamic display part
    M5.Lcd.setCursor(0, 51);

    // Connect to WiFi
    StaticJsonDocument<512> n_jsondata;

    int retryCount = 0;
    const int maxRetry = 5;
    while (SD.begin(GPIO_NUM_4) != true && retryCount < maxRetry) {
        M5.Lcd.println("SD Card Mount Failed");
        delay(500);
        retryCount++;
    }
    if (retryCount >= maxRetry) {
        M5.Lcd.println("Failed to mount SD card after maximum retries");
        return {"", ""};  // Return empty strings to indicate failure
    }

    Serial.println("microSD card initialized.");

    if (SD.exists("/RFID.txt")) {
        Serial.println("RFID.txt exists.");
        delay(500);
        File f = SD.open("/RFID.txt", FILE_READ);

        if (f) {
            while (f.available())
            {
                JsonData.concat(f.readString());
            }
            f.close();
            sdstat = 1;
        } else {
            M5.Lcd.println("error opening /RFID.txt");
            sdstat = 0;
        }
    } else {
        M5.Lcd.println("RFID.txt doesn't exit.");
        Serial.println("RFID.txt doesn't exit.");
        sdstat = 0;
    }

    String i_host;
    JsonArray i_ssids;
    if (sdstat == 1) {
        DeserializationError error = deserializeJson(n_jsondata, JsonData);

        if (error)
        {
            M5.Lcd.print(F("deserializeJson() failed: "));
            M5.Lcd.println(error.f_str());
        }
        else
        {
            i_ssids = n_jsondata["ssids"].as<JsonArray>();
            i_host = n_jsondata["host"].as<String>();

            Serial.println("Can read from JSON Data!");
            for (JsonVariant ssidVariant : i_ssids)
            {
                String ssid = ssidVariant["ssid"].as<String>();
                Serial.printf("ssid: %s\n", ssid);
                Serial.println("pass: <masked>");
            }
            Serial.printf("host: %s\n", i_host);
        }

        const size_t MAX_SSIDS = 5;
        if (!n_jsondata.containsKey("ssids") || !n_jsondata.containsKey("host")) {
            M5.Lcd.println("Missing required fields in config");
            return {"", ""};
        }

        JsonArray ssids = n_jsondata["ssids"].as<JsonArray>();
        if (ssids.size() == 0 || ssids.size() > MAX_SSIDS) {
            M5.Lcd.println("Invalid number of SSIDs");
            return {"", ""};
        }
        // try to connect to the first available network
        for (JsonVariant ssidVariant : i_ssids)
        {
            if (!ssidVariant.containsKey("ssid") || !ssidVariant.containsKey("pass")) {
                M5.Lcd.println("Missing SSID or password");
                return {"", ""};
            }

            String ssid = ssidVariant["ssid"].as<String>();
            String pass = ssidVariant["pass"].as<String>();

            char buf_ssid[33], buf_pass[65];
            ssid.toCharArray(buf_ssid, 33);
            pass.toCharArray(buf_pass, 65);

            WiFi.begin(buf_ssid, buf_pass);
            M5.Lcd.printf("Connecting to %s\n", ssid.c_str());
            Serial.printf("Connecting to %s\n", ssid.c_str());

            unsigned long startTime = millis();
            const unsigned long timeout = 10000; // 10 seconds timeout
            while (WiFi.status() != WL_CONNECTED && (millis() - startTime < timeout))
            {
                delay(500);
                M5.Lcd.print(".");
            }
            if (WiFi.status() == WL_CONNECTED) {
                M5.Lcd.println("");
                M5.Lcd.println("WiFi connected");
                Serial.println("WiFi connected");
                i_ssid = ssid; // Save the connected SSID
                break;
            } else {
                M5.Lcd.println("\nWiFi connection timeout");
            }
        }

        if (WiFi.status() != WL_CONNECTED) {
            M5.Lcd.println("Failed to connect to any network");
            Serial.println("Failed to connect to any network");
            return {"", ""};  // Return empty strings to indicate failure
        }

        M5.Lcd.println("");
        M5.Lcd.println("WiFi connected");
        Serial.println("WiFi connected");
    }

    return {i_ssid, i_host};
}