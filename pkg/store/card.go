package store

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/whywaita/poker-go"
	"github.com/whywaita/rfid-poker/pkg/query"
)

func GetCardBySerial(ctx context.Context, conn *sql.DB, serial string) ([]poker.Card, error) {
	q := query.New(conn)
	cards, err := q.GetCardBySerial(ctx, serial)
	if err != nil {
		return nil, fmt.Errorf("q.GetCardBySerial(): %w", err)
	}

	result := make([]poker.Card, 0, len(cards))
	for _, c := range cards {
		card := poker.Card{
			Rank: poker.UnmarshalRankString(c.CardRank),
			Suit: poker.UnmarshalSuitString(c.CardSuit),
		}
		result = append(result, card)
	}

	return result, nil
}

func AddCard(ctx context.Context, conn *sql.DB, card poker.Card, serial string) error {
	// Get or create current game
	gameID, err := GetOrCreateCurrentGame(ctx, conn)
	if err != nil {
		return fmt.Errorf("GetOrCreateCurrentGame(): %w", err)
	}

	q := query.New(conn)
	_, err = q.AddCard(ctx, query.AddCardParams{
		Serial:   serial,
		CardSuit: card.Suit.String(),
		CardRank: card.Rank.String(),
		GameID:   sql.NullString{String: gameID, Valid: true},
		IsBoard:  false,
	})
	if err != nil {
		return fmt.Errorf("q.AddCard(): %w", err)
	}

	slog.InfoContext(ctx, "Added card",
		slog.String("game_id", gameID),
		slog.String("event", "card_added"),
		slog.String("card_rank", card.Rank.String()),
		slog.String("card_suit", card.Suit.String()),
		slog.String("serial", serial))
	return nil
}
