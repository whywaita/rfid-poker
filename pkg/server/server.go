package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/whywaita/poker-go"
	"github.com/whywaita/rfid-poker/pkg/playercards"
	"github.com/whywaita/rfid-poker/pkg/reader"
	"golang.org/x/net/websocket"
)

func Run(ctx context.Context, configPath string) error {
	go func() {
		runtime.GOMAXPROCS(runtime.NumCPU())
		log.Println(http.ListenAndServe(":6060", nil))
	}()

	handCh := make(chan playercards.HandData)
	deviceCh := make(chan reader.Data)
	updatedCh := make(chan struct{})

	cardConfigs, err := playercards.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("playercards.LoadConfig(%s): %w", configPath, err)
	}

	go func() {
		if err := reader.PollingDevices(deviceCh); err != nil {
			log.Printf("reader.PollingDevices(): %v", err)
			return
		}
	}()
	go func() {
		log.Printf("Start loading cards...")
		if err := playercards.LoadCardsWithChannel(*cardConfigs, 2, handCh, deviceCh); err != nil {
			log.Printf("playercards.LoadCardsWithChannel(ctx): %v", err)
			return
		}
	}()
	go func() {
		if err := ReceiveData(handCh, updatedCh); err != nil {
			log.Printf("ReceiveData(): %v", err)
			return
		}
	}()

	//m := NewMux(handCh)
	e := echo.New()
	e.Use(middleware.Logger())
	e.GET("/ws", func(c echo.Context) error {
		return ws(c, updatedCh)
	})
	if err := e.Start(":8080"); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Send is struct for SSE
type Send struct {
	Players []SendPlayer `json:"players"`
}

type SendPlayer struct {
	Name   string     `json:"name"`
	Hand   []SendCard `json:"hand"`
	Equity float64    `json:"equity"`
}

type SendCard struct {
	Suit string `json:"suit"`
	Rank string `json:"rank"`
}

func ReceiveData(ch chan playercards.HandData, updateCh chan struct{}) error {
	for v := range ch {
		log.Printf("receive card in server: %s", v)
		if len(v.Cards) != 2 {
			log.Printf("invalid card: %s", v)
			continue
		}

		_, err := AddPlayer(poker.Player{
			Name:  v.SerialNumber, // TODO: convert serial number to name
			Hand:  v.Cards,
			Score: 0,
		})
		if err != nil {
			log.Printf("AddPlayer(): %v", err)
			continue
		}
		updateCh <- struct{}{}
	}

	return nil
}

func ws(c echo.Context, ch chan struct{}) error {
	websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()

		if err := sendPlayer(ws); err != nil {
			c.Logger().Errorf(err.Error())
		}

		for {
			<-ch
			err := sendPlayer(ws)
			if err != nil {
				c.Logger().Errorf(err.Error())
			}
		}
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}

func sendPlayer(ws *websocket.Conn) error {
	send, err := getSend()
	if err != nil {
		return fmt.Errorf("getSend(): %w", err)
	}

	b, err := json.Marshal(send)
	if err != nil {
		return fmt.Errorf("json.Marshal(%v): %w", send, err)
	}

	log.Println("Send: ", string(b))
	if err := websocket.Message.Send(ws, string(b)); err != nil {
		return fmt.Errorf("websocket.Message.Send(): %w", err)
	}
	return nil
}

func getSend() (*Send, error) {
	send := &Send{}
	data := GetStored()

	for _, s := range data {
		hand := make([]SendCard, 0, len(s.Player.Hand))

		for _, card := range s.Player.Hand {
			hand = append(hand, SendCard{
				Suit: card.Suit.String(),
				Rank: card.Rank.String(),
			})
		}

		send.Players = append(send.Players, SendPlayer{
			Name:   s.Player.Name,
			Hand:   hand,
			Equity: s.Equity,
		})
	}

	return send, nil
}
