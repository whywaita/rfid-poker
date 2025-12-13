# Test Wired Client

A software-only simulator for testing the RFID Poker server without physical hardware (M5Stack + RFID).

## Overview

The `test-wired-client` is a command-line tool that simulates the behavior of the hardware wired-client. It allows you to:

- Send card data to the server interactively
- Test server functionality without RFID hardware
- Simulate different antenna types (player, board, muck)
- Use card names (e.g., `As`, `Kh`) or UIDs directly

## Building

```bash
make bin/test-wired-client
```

## Usage

### Basic Usage

```bash
./bin/test-wired-client --serial player1 --type player --server http://localhost:8080
```

### Command-Line Options

- `--config` - Path to config.yaml file (default: `./config.yaml`)
- `--serial` - Antenna serial/device ID (required)
- `--type` - Antenna type: player, board, or muck (default: `player`)
- `--pair-id` - Antenna pair ID (default: `1`)
- `--server` - Server URL for HTTP POST (default: `http://localhost:8080`)
- `--debug` - Enable debug logging

### Configuration File

The test client uses the same `config.yaml` file as the server, which maps card UIDs to card names:

```yaml
card_ids:
  040e3bd2286b85: As  # Ace of Spades
  042f43d2286b85: Kh  # King of Hearts
  051a1b9a776b85: 2d  # Two of Diamonds
  # ... more cards
```

See `config.yaml.example` for a complete example with all 52 cards.

## Interactive Commands

Once the client is running, you can use the following commands:

### Send Card by Name

Enter a card name using standard notation:
- Ranks: `A`, `K`, `Q`, `J`, `T` (10), `9`, `8`, `7`, `6`, `5`, `4`, `3`, `2`
- Suits: `s` (spades), `h` (hearts), `d` (diamonds), `c` (clubs)

```
> As    # Send Ace of Spades
✅ Card sent: As (UID: 040e3bd2286b85)

> Kh    # Send King of Hearts
✅ Card sent: Kh (UID: 042f43d2286b85)
```

Card names are case-insensitive (`as`, `AS`, `As` all work).

### Send Card by UID

Enter the card UID directly:

```
> 040e3bd2286b85
✅ Card sent: As (UID: 040e3bd2286b85)
```

### List Available Cards

```
> list

Available cards:
================

♠ Spades:
  As   -> 040e3bd2286b85
  Ks   -> 040f43d2286b85
  ...

♥ Hearts:
  Ah   -> 042e3bd2286b85
  ...
```

### Exit

```
> quit
```

or

```
> exit
```

## Example Usage Scenarios

### Simulating Player 1 Antenna

```bash
./bin/test-wired-client --serial player1 --type player --server http://localhost:8080

> As
✅ Card sent: As (UID: 040e3bd2286b85)
> Kh
✅ Card sent: Kh (UID: 042f43d2286b85)
> quit
```

### Simulating Board Antenna

```bash
./bin/test-wired-client --serial board1 --type board --server http://localhost:8080

> Ah
✅ Card sent: Ah (UID: 042e3bd2286b85)
> Kd
✅ Card sent: Kd (UID: 050f43d2286b85)
> Qc
✅ Card sent: Qc (UID: 06101b9a776b85)
```

### Simulating Muck Antenna

```bash
./bin/test-wired-client --serial muck1 --type muck --server http://localhost:8080
```

## API Endpoint

The test client sends POST requests to `/card` with the following JSON body:

```json
{
  "uid": "040e3bd2286b85",
  "device_id": "player1",
  "pair_id": 1
}
```

This matches the same format used by the hardware wired-client.

## Troubleshooting

### Error: "serial is required"

You must specify the `--serial` flag:

```bash
./bin/test-wired-client --serial player1
```

### Error: "failed to load config"

Make sure the config.yaml file exists and is properly formatted:

```bash
./bin/test-wired-client --config /path/to/config.yaml --serial player1
```

### Error: "Unknown card"

Use the `list` command to see all available cards, or check your config.yaml file.

## Related

- [wired-client](./wired-client-architecture.md) - Hardware client documentation
- [config.yaml.example](../config.yaml.example) - Example configuration file
