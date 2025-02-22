# Equity viewer for poker (No-limit Texas Hold'em)

![demo](./doc/media/demo.gif)

## Requirements

- M5Stack + RFID module
  - Players x N (N is the number of players) + 1 (for muck) + 1 (for board)
  - We tested with...
    - [M5Stack Core2](https://docs.m5stack.com/en/core/core2)
    - [Unit RFID2](https://docs.m5stack.com/en/unit/rfid2)
      - This unit has been confirmed by [Switch Science](https://www.switch-science.com/products/8301) to be compliant with Japan's Radio Law. ([ref](https://mag.switch-science.com/2022/05/24/m5stack-3/))
- Player cards with NFC chip (ISO/IEC 14443 Type A)
  - We tested player cards include MIFARE Ultralight EV1

## Setup

### Prepare a config file

```bash
$ cat config.yaml
card_ids:  ## UID of NEC card
  040e3bd2286b85: As
  040f43d2286b85: Qc
  04101b9a776b85: Kc
  ...
```

### Run the server

Execute binary with environment variables.

```bash
# Set the path to the config file
# You can set path as `http://` or `https://` to get the config file from the server.
export RFID_POKER_CONFIG_PATH="./config.yaml"

# Set MySQL connection information
export RFID_POKER_MYSQL_USER=<your_mysql_user>
export RFID_POKER_MYSQL_PASS=<your_mysql_password>
export RFID_POKER_MYSQL_HOST=<your_mysql_host>
export RFID_POKER_MYSQL_PORT=<your_mysql_port>
export RFID_POKER_MYSQL_DATABASE=<your_mysql_database>

# Run the server
$ go run main.go
```

## Components

### Server

The server is a golang application that runs on a server.

#### `GET /ws` (websocket)

The server will upgrade the connection to a websocket. The server send an info about players to the client.

The body of the message is as follows:

```json
{
  "boards": [
    {
      "rank": "A",
      "suit": "hearts"
    },
    {
      "rank": "K",
      "suit": "hearts"
    },
    {
      "rank": "Q",
      "suit": "hearts"
    },
    {
      "rank": "J",
      "suit": "hearts"
    },
    {
      "rank": "T",
      "suit": "hearts"
    }
  ],
  "players": [
    {
      "name": "Player 1",
      "hand": [
        {
          "rank": "A",
          "suit": "spades"
        },
        {
          "rank": "K",
          "suit": "spades"
        }
      ],
      "equity": 0.5
    },
    {
      "name": "Player 2",
      "cards": [
        {
          "rank": "A",
          "suit": "clubs"
        },
        {
          "rank": "K",
          "suit": "clubs"
        }
      ],
      "equity": 0.5
    }
  ]
}
```

#### POST /device/boot

The server will send a message to the device to boot.

```json
{
  "device_id": "device_id",  // as Mac address (in M5stack)
  "pair_ids": [1, 2, 3, ...] // antenna pair ids
}
```

#### POST /card

The server will send a message to the device to read a card.

```json
{
  "device_id": "device_id",    // as Mac address (in M5stack)
  "pair_id": 1,                // antenna pair id
  "card_id": "040e3bd2286b85"  // as UID of NFC card
}
```

### ui

Ths ui is a Next.js application that runs on a client.

[ui](./ui) directory is a Next.js application.

You can use the newest code in GitHub Pages ([https://whywaita.github.io/rfid-poker/](https://whywaita.github.io/rfid-poker/)).

### Client

This is a M5Stack application that runs on a M5Stack device.

[client/m5stack](./client/m5stack) directory is a M5Stack application.