package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sort"
	"sync"

	"github.com/whywaita/rfid-poker/pkg/query"
	"github.com/whywaita/rfid-poker/pkg/store"

	"github.com/coder/websocket"
	"github.com/labstack/echo/v4"
)

// WebSocketManager manages WebSocket connections
type WebSocketManager struct {
	mu       sync.Mutex
	clients  map[*websocket.Conn]struct{}
	notifyCh chan struct{}
}

var wsManager = &WebSocketManager{
	clients:  make(map[*websocket.Conn]struct{}),
	notifyCh: make(chan struct{}, 1000), // with buffer
}

func (m *WebSocketManager) addClient(ws *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients[ws] = struct{}{}
}

func (m *WebSocketManager) removeClient(ws *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clients, ws)
}

// broadcast sends a message to all WebSocket clients
func (m *WebSocketManager) broadcast(q *query.Queries) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for client := range m.clients {
		go func(ws *websocket.Conn) {
			if err := sendPlayer(context.Background(), q, ws); err != nil {
				log.Printf("Failed to send update to WebSocket: %v", err)
			}
		}(client)
	}
}

// Send is struct for WebSocket sending
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

	wsConn, err := websocket.Accept(c.Response(), c.Request(), &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		return fmt.Errorf("failed to accept WebSocket: %w", err)
	}
	defer wsConn.Close(websocket.StatusNormalClosure, "")

	wsManager.addClient(wsConn)
	defer wsManager.removeClient(wsConn)

	ctx := c.Request().Context()

	if err := sendPlayer(ctx, q, wsConn); err != nil {
		c.Logger().Errorf(err.Error())
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-wsManager.notifyCh:
			wsManager.broadcast(q)
		}
	}
}

func notifyClients() {
	select {
	case wsManager.notifyCh <- struct{}{}:
	default:
		// skip if the channel is full
	}
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
	w, err := ws.Writer(ctx, websocket.MessageText)
	if err != nil {
		return fmt.Errorf("ws.Writer(): %w", err)
	}

	if _, err := io.Copy(w, bytes.NewBuffer(b)); err != nil {
		return fmt.Errorf("io.Copy(): %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("w.Close(): %w", err)
	}

	return nil
}

func getSend(ctx context.Context, q *query.Queries) (*Send, error) {
	send := &Send{}
	data, err := store.GetStored(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("GetStored(): %w", err)
	}

	board, err := store.GetBoard(ctx, q)
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

		sort.SliceStable(hand, func(i, j int) bool {
			return hand[i].Rank < hand[j].Rank
		})

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
