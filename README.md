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

```bash
# Run the server
$ go run main.go
```

### Develop

```bash
$ ENV=development go run main.go
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
