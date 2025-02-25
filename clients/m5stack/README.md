# m5stack-client

This is a client for the M5Stack device. It is written in C++.

We developed with PlatformIO.

## Configure (M5Stack Core2)

1. Put `/RFID.txt` file in the root of the SD card (as TF card).

```json
{
  "ssids": [
    {
      "ssid": "your-ssid-1",
      "pass": "your-password-2"
    },
    {
      "ssid": "your-ssid-2",
      "pass": "your-password-2"
    }
  ],
  "host": "https://your-host" // your server address
}
```

The device will connect to the SSID in the order of the list.

## Configure (M5Stack Atom)

M5stack Atom is supported only one SSID.

1. Build the project with the following command.

Don't forget to escape the double quotes in the API_HOST.

```bash
WIFI_SSID="your-ssid" WIFI_PASSWORD="your-password" API_HOST='\"https\://your-host.example.com\"' pio run -t upload --environment m5stack-atom
```

