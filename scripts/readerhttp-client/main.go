package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/whywaita/rfid-poker/pkg/readerhttp"
)

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	var (
		inSerial = flag.String("serial", "", "serial number of device")
		inCards  = flag.String("cards", "", "card uid, comma separated (ex: 12345678, 9875432)")
		inHost   = flag.String("host", "localhost", "host of server")
	)
	flag.Parse()

	if *inSerial == "" {
		return fmt.Errorf("-serial is required")
	}
	if *inCards == "" {
		return fmt.Errorf("-cards is required")
	}
	if *inHost == "" {
		return fmt.Errorf("-host is required")
	}
	cards := strings.Split(*inCards, ",")
	// if len(cards) != 2 && len(cards) != 3 {
	// 	return fmt.Errorf("invalid cards: %s", *inCards)
	// }
	u, err := url.Parse(*inHost)
	if err != nil {
		return fmt.Errorf("url.Parse(): %w", err)
	}

	ctx := context.Background()
	if err := DoReq(ctx, *inSerial, cards, u); err != nil {
		return fmt.Errorf("doReq(): %w", err)
	}

	return nil
}

// DoReq send card data to server
func DoReq(ctx context.Context, serial string, cards []string, u *url.URL) error {
	for _, card := range cards {
		body := readerhttp.Card{
			UID:    card,
			Serial: serial,
		}
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("json.Marshal(): %w", err)
		}

		if err := doReq(ctx, b, u); err != nil {
			return fmt.Errorf("doReq(): %w", err)
		}
		time.Sleep(1 * time.Second)
	}

	return nil
}

func doReq(ctx context.Context, body []byte, u *url.URL) error {
	log.Println("doReq()")
	u = u.JoinPath(u.Path, "card")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("http.NewRequestWithContext(): %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("client.Do(): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid status code: %d", resp.StatusCode)
	}
	log.Println("resp.StatusCode: ", resp.StatusCode)

	return nil
}
