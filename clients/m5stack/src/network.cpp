#include <ArduinoJson.h>
#include <WiFi.h>
#include "SD.h"
#include <M5Unified.h>

String JsonData;
int sdstat = 0;
String i_ssid, i_pass;

std::tuple<String, String> setupNetwork() {
    // Set the cursor to the dynamic display part
    M5.Lcd.setCursor(0, 51);

    // Connect to WiFi
    StaticJsonDocument<512> n_jsondata;
    while (SD.begin(GPIO_NUM_4) != true) {
        M5.Lcd.println("SD Card Mount Failed");
        delay(500);
    }

    Serial.println("microSD card initialized.");

    if (SD.exists("/SSID.txt")) {
        Serial.println("SSID.txt exists.");
        delay(500);
        File f = SD.open("/SSID.txt", FILE_READ);

        if (f) {
            while (f.available())
            {
                JsonData.concat(f.readString());
            }
            f.close();
            sdstat = 1;
        } else {
            M5.Lcd.println("error opening /SSID.txt");
            sdstat = 0;
        }
    } else {
        M5.Lcd.println("SSID.txt doesn't exit.");
        Serial.println("SSID.txt doesn't exit.");
        sdstat = 0;
    }

    String i_host;
    if (sdstat == 1) {
        DeserializationError error = deserializeJson(n_jsondata, JsonData);

        if (error)
        {
            M5.Lcd.print(F("deserializeJson() failed: "));
            M5.Lcd.println(error.f_str());
        }
        else
        {
            i_ssid = n_jsondata["ssid"].as<String>();
            i_pass = n_jsondata["pass"].as<String>();
            i_host = n_jsondata["host"].as<String>();

            Serial.println("Can read from JSON Data!");
            Serial.printf("ssid: %s\n", i_ssid);
            Serial.println("pass: <masked>");
            Serial.printf("host: %s\n", i_host);
        }

        char buf_ssid[33], buf_pass[65];
        i_ssid.toCharArray(buf_ssid, 33);
        i_pass.toCharArray(buf_pass, 65);

        WiFi.begin(buf_ssid, buf_pass);
        M5.Lcd.printf("Connecting to %s\n", i_ssid);
        Serial.printf("Connecting to %s\n", i_ssid);
        while (WiFi.status() != WL_CONNECTED)
        {
            delay(500);
            M5.Lcd.print(".");
        }

        M5.Lcd.println("");
        M5.Lcd.println("WiFi connected");
        Serial.println("WiFi connected");
    }

    return {i_ssid, i_host};
}