package store

import (
	"context"
	"database/sql"
	"fmt"
	"log"
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

func calcEquity(ctx context.Context, q *query.Queries, updatedCh chan struct{}) error {
	calcEquityMu.Lock()
	defer calcEquityMu.Unlock()
	playersRow, err := q.GetPlayersWithHand(ctx)
	if err != nil {
		return fmt.Errorf("db.GetPlayersWithHand(): %w", err)
	}

	players := make([]poker.Player, 0, len(playersRow))

	// check if one of the players has equity zero
	hasEquityZero := false
	for _, p := range playersRow {
		if !p.Equity.Valid && !hasEquityZero {
			hasEquityZero = true
		}
		pgCardA, err := query.Card{
			Suit:    p.CardASuit,
			Rank:    p.CardARank,
			IsBoard: p.CardAIsBoard,
		}.ToPokerGo()
		if err != nil {
			return fmt.Errorf("cardA.ToPokerGo(): %w", err)
		}
		pgCardB, err := query.Card{
			Suit:    p.CardBSuit,
			Rank:    p.CardBRank,
			IsBoard: p.CardBIsBoard,
		}.ToPokerGo()
		if err != nil {
			return fmt.Errorf("cardB.ToPokerGo(): %w", err)
		}

		players = append(players, poker.Player{
			Name: p.Name,
			Hand: []poker.Card{
				*pgCardA,
				*pgCardB,
			},
		})
	}

	if hasEquityZero {
		// if one of the players has equity zero, need to calculate equity
		// So will reset all equity
		if err := q.ResetEquity(ctx); err != nil {
			return fmt.Errorf("db.ResetEquity(): %w", err)
		}
		updatedCh <- struct{}{}
	}

	if len(players) <= 1 {
		// if players is less than 2, no need to calculate equity
		return nil
	}

	board, err := GetBoard(ctx, q)
	if err != nil {
		return fmt.Errorf("GetBoard(): %w", err)
	}

	log.Printf("Start EvaluateEquityByMadeHandWithCommunity(%+v, %+v)", players, board)
	equities, err := poker.EvaluateEquityByMadeHandWithCommunity(players, board)
	if err != nil {
		return fmt.Errorf("poker.EvaluateEquityByMadeHandWithCommunity: %w", err)
	}
	log.Println("End EvaluateEquityByMadeHand")

	for i, p := range playersRow {
		if err := q.UpdateEquity(ctx, query.UpdateEquityParams{
			Equity: sql.NullFloat64{
				Float64: equities[i],
				Valid:   true,
			},
			ID: p.HandID,
		}); err != nil {
			return fmt.Errorf("db.UpdatePlayerEquity(hand_id: %v): %w", p.HandID, err)
		}
	}

	updatedCh <- struct{}{}

	return nil
}

//func isStoredCard(ctx context.Context, q *query.Queries, card poker.Card) (bool, error) {
//	_, err := q.GetCardByRankSuit(ctx, query.GetCardByRankSuitParams{
//		Rank: card.Rank.String(),
//		Suit: card.Suit.String(),
//	})
//	if err != nil {
//		if errors.Is(err, sql.ErrNoRows) {
//			return false, nil
//		}
//		return false, fmt.Errorf("db.GetCardByRankSuit(): %w", err)
//	}
//	return true, nil
//}

func ClearGame(ctx context.Context, conn *sql.DB) error {
	db := query.New(conn)

	if err := db.DeleteHandAll(ctx); err != nil {
		return fmt.Errorf("db.DeleteHandAll(): %w", err)
	}

	if err := db.DeleteCardAll(ctx); err != nil {
		return fmt.Errorf("db.DeleteCardAll(): %w", err)
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
