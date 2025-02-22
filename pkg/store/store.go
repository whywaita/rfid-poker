package store

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/whywaita/poker-go"
	"github.com/whywaita/rfid-poker/pkg/query"
)

type Stored struct {
	PlayerName string
	Hand       []poker.Card
	Equity     float64
}

var (
	calcEquityMu sync.RWMutex
)

func ClearGame(ctx context.Context, conn *sql.DB) error {
	db := query.New(conn)

	if err := db.DeleteCardAll(ctx); err != nil {
		return fmt.Errorf("db.DeleteCardAll(): %w", err)
	}

	if err := db.DeleteHandAll(ctx); err != nil {
		return fmt.Errorf("db.DeleteHandAll(): %w", err)
	}

	return nil
}

func GetStored(ctx context.Context, q *query.Queries) ([]Stored, error) {
	players, err := q.GetPlayersWithHand(ctx)
	if err != nil {
		return nil, fmt.Errorf("q.GetPlayersWithHand(): %w", err)
	}
	stored := make([]Stored, 0, len(players))

	for _, p := range players {
		cardA := poker.Card{
			Suit: poker.UnmarshalSuitString(p.CardASuit),
			Rank: poker.UnmarshalRankString(p.CardARank),
		}
		cardB := poker.Card{
			Suit: poker.UnmarshalSuitString(p.CardBSuit),
			Rank: poker.UnmarshalRankString(p.CardBRank),
		}

		stored = append(stored, Stored{
			PlayerName: p.Name,
			Hand: []poker.Card{
				cardA,
				cardB,
			},
			Equity: p.Equity.Float64,
		})
	}

	return stored, nil
}
