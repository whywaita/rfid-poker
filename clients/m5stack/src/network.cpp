#include <ArduinoJson.h>
#include <WiFi.h>

std::tuple<String, String> setupNetwork() {
// Use build_args to define these values during compilation
// Example: platformio.ini should contain:
// build_flags =
//     -D WIFI_SSID=\"your_ssid\"
//     -D WIFI_PASSWORD=\"your_password\"
//     -D API_HOST=\"http://your-api.example.com\"

// Define string literals for credentials
// These will be replaced by the compiler if defined via build_flags
#define STRINGIFY(x) #x
#define TOSTRING(x) STRINGIFY(x)

#ifdef WIFI_SSID
  const char *ssid = TOSTRING(WIFI_SSID);
#else
  const char *ssid = "";
  Serial.println("Warning: WIFI_SSID not defined in build_args");
#endif

#ifdef WIFI_PASSWORD
  const char *password = TOSTRING(WIFI_PASSWORD);
#else
  const char *password = "";
  Serial.println("Warning: WIFI_PASSWORD not defined in build_args");
#endif

#ifdef API_HOST
  Serial.println("API_HOST defined as: " + String(TOSTRING(API_HOST)));
  const char *host = TOSTRING(API_HOST);
#else
  const char *host = "";
  Serial.println("Warning: API_HOST not defined in build_args");
#endif

  Serial.println("ATOM WiFi Setup");
  Serial.printf("Connecting to %s\n", ssid);

  // Create a proper String object from the host
  String hostStr = String(host);
  // Remove the double quotes (prefix and suffix)
  hostStr.remove(0, 1);
  hostStr.remove(hostStr.length() - 1);
  Serial.printf("Host: %s\n", hostStr.c_str());

  WiFi.begin(ssid, password);

  unsigned long startTime = millis();
  const unsigned long timeout = 10000; // 10 seconds timeout

  while (WiFi.status() != WL_CONNECTED && (millis() - startTime < timeout)) {
    delay(500);
    Serial.print(".");
  }

  if (WiFi.status() == WL_CONNECTED) {
    Serial.println("");
    Serial.println("WiFi connected");
    Serial.printf("IP address: %s\n", WiFi.localIP().toString().c_str());
    return {String(ssid), hostStr}; // Return the fixed host string
  }

  Serial.println("");
  Serial.println("WiFi connection failed");
  return {"", ""}; // Return empty strings to indicate failure
}