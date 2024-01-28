package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/whywaita/rfid-poker/pkg/query"

	"github.com/labstack/echo/v4"
	"golang.org/x/net/websocket"
)

// Send is struct for SSE
type Send struct {
	Players []SendPlayer `json:"players"`
	Board   []SendCard   `json:"board"`
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

func ws(c echo.Context, conn *sql.DB, notifyCh chan struct{}) error {
	q := query.New(conn)
	websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()
		ctx := c.Request().Context()

		if err := sendPlayer(ctx, q, ws); err != nil {
			c.Logger().Errorf(err.Error())
		}

		for {
			<-notifyCh
			err := sendPlayer(ctx, q, ws)
			if err != nil {
				c.Logger().Errorf(err.Error())
			}
		}
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}

func sendPlayer(ctx context.Context, q *query.Queries, ws *websocket.Conn) error {
	send, err := getSend(ctx, q)
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

func getSend(ctx context.Context, q *query.Queries) (*Send, error) {
	send := &Send{}
	data, err := GetStored(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("GetStored(): %w", err)
	}

	board, err := GetBoard(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("GetBoard(): %w", err)
	}

	for _, s := range data {
		hand := make([]SendCard, 0, len(s.Hand))

		for _, card := range s.Hand {
			hand = append(hand, SendCard{
				Suit: card.Suit.String(),
				Rank: card.Rank.String(),
			})
		}

		send.Players = append(send.Players, SendPlayer{
			Name:   s.PlayerName,
			Hand:   hand,
			Equity: s.Equity,
		})
	}

	for _, card := range board {
		send.Board = append(send.Board, SendCard{
			Suit: card.Suit.String(),
			Rank: card.Rank.String(),
		})
	}

	return send, nil
}
