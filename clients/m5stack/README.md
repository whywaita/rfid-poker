# m5stack-client

This is a client for the M5Stack device. It is written in C++.

We developed with PlatformIO.

## Configure

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