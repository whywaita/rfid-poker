package store

import (
	"context"
	"database/sql"
	"fmt"

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
			Rank: poker.UnmarshalRankString(c.Rank),
			Suit: poker.UnmarshalSuitString(c.Suit),
		}
		result = append(result, card)
	}

	return result, nil
}

func AddCard(ctx context.Context, conn *sql.DB, card poker.Card, serial string) error {
	q := query.New(conn)
	_, err := q.AddCard(ctx, query.AddCardParams{
		Serial: serial,
		Suit:   card.Suit.String(),
		Rank:   card.Rank.String(),
	})
	if err != nil {
		return fmt.Errorf("q.AddCard(): %w", err)
	}

	return nil
}
