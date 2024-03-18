# Equity viewer for poker (No-limit Texas Hold'em)

![demo](./doc/media/demo.gif)

## Requirements

- PaSoRi
  - Players x N (N is the number of players) + 1 (for muck)
  - We tested with [RC-S380](https://www.sony.co.jp/Products/felica/consumer/products/RC-S380.html)
- Player cards with NFC chip (ISO/IEC 14443 Type A)
  - We tested player cards include MIFARE Ultralight EV1

## Setup

### Install a dependencies

- libusb-dev

### Unload the kernel driver for the device

```bash
$ sudo echo "blacklist port100" >> /etc/modprobe.d/noport100.conf
```

### Prepare a config file

```bash
$ cat config.yaml
card_ids:  ## UID of NEC card
  040e3bd2286b85000000: As
  040f43d2286b85000000: Qc
  04101b9a776b85000000: Kc
  ...
players:  ## The serial of PaSoRi for each player
  - name: "Player 1"
    serial: 0000000
  - name: "Player 2"
    serial: 0000001
muck_serial: 0000002  ## The serial of PaSoRi for muck
board_serial: 0000003

# Optional values

http_mode: true  ## If true, the server will receive the cards from the HTTP request (default: false)
```

### Run the server

```bash
# Run the server with root
$ sudo go run main.go

# Or run the server with usb group
$ sudo gpasswd -a $USER usb
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
