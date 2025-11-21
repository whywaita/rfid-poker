# m5stack-client

This is a client for the M5Stack Atom device. It is written in C++.

We developed with PlatformIO.

## Configure

M5Stack Atom is supported with build-time configuration.

### Build-time Configuration

The following environment variables are required:

- `WIFI_SSID`: WiFi SSID to connect
- `WIFI_PASSWORD`: WiFi password
- `API_HOST`: Server API endpoint (must be escaped, e.g., `'\"https\://your-host.example.com\"'`)
- `CLIENT_TYPE`: Type of client (optional, but recommended)
  - `player`: Player mode - 2 RFID readers for hole cards (sends cards only when both are detected)
  - `board`: Board mode - 5 RFID readers for community cards (sends each card immediately)
  - `muck`: Muck mode - 2 RFID readers for muck cards (sends cards only when both are detected)
  - If not specified, defaults to 2 RFID readers

### Build and Upload

Build the project with the following command:

```bash
WIFI_SSID="your-ssid" WIFI_PASSWORD="your-password" API_HOST='\"https\://your-host.example.com\"' CLIENT_TYPE="player" pio run -t upload --environment m5stack-atom
```

### Examples

For player device:
```bash
WIFI_SSID="your-ssid" WIFI_PASSWORD="your-password" API_HOST='\"https\://your-host.example.com\"' CLIENT_TYPE="player" pio run -t upload --environment m5stack-atom
```

For board device:
```bash
WIFI_SSID="your-ssid" WIFI_PASSWORD="your-password" API_HOST='\"https\://your-host.example.com\"' CLIENT_TYPE="board" pio run -t upload --environment m5stack-atom
```

