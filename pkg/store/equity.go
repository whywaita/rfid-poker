package store

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/whywaita/poker-go"
	"github.com/whywaita/rfid-poker/pkg/query"
)

// CalcEquity calculate equity of players
func CalcEquity(ctx context.Context, q *query.Queries) error {
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
			CardSuit: p.CardASuit,
			CardRank: p.CardARank,
			IsBoard:  p.CardAIsBoard,
		}.ToPokerGo()
		if err != nil {
			return fmt.Errorf("cardA.ToPokerGo(): %w", err)
		}
		pgCardB, err := query.Card{
			CardSuit: p.CardBSuit,
			CardRank: p.CardBRank,
			IsBoard:  p.CardBIsBoard,
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
	}

	if len(players) <= 1 {
		// if players is less than 2, no need to calculate equity
		if err := q.ResetEquity(ctx); err != nil {
			return fmt.Errorf("db.ResetEquity(): %w", err)
		}
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
			Equity: sql.NullFloat64{Float64: equities[i], Valid: true},
			ID:     p.HandID,
		}); err != nil {
			return fmt.Errorf("db.UpdatePlayerEquity(hand_id: %v): %w", p.HandID, err)
		}
	}

	return nil
}
