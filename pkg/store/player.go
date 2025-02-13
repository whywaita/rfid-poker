package store

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/whywaita/poker-go"
	"github.com/whywaita/rfid-poker/pkg/query"
)

func AddHand(ctx context.Context, conn *sql.DB, input []poker.Card, serial string) error {
	if len(input) != 2 {
		return fmt.Errorf("invalid input length (not 2): %v", input)
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("conn.BeginTx(): %w", err)
	}
	defer func() {
		if r := recover(); r != nil || err != nil {
			tx.Rollback()
		}
	}()
	qWithTx := query.New(tx)

	sort.SliceStable(input, func(i, j int) bool {
		return input[i].Rank < input[j].Rank
	})

	player, err := qWithTx.GetPlayerBySerial(ctx, serial)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("q.GetPlayerBySerial(): %w", err)
	}

	hand, err := qWithTx.AddHand(ctx, player.ID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("db.AddHand(): %w", err)
	}

	for _, c := range input {
		_, err = qWithTx.AddCard(ctx, query.AddCardParams{
			Suit:    c.Suit.String(),
			Rank:    c.Rank.String(),
			Serial:  serial,
			IsBoard: false,
		})
		if err != nil && !strings.HasPrefix(err.Error(), "UNIQUE constraint failed:") {
			tx.Rollback()
			return fmt.Errorf("q.AddCard(): %w", err)
		}

		dbCard, err := qWithTx.GetCardByRankSuit(ctx, query.GetCardByRankSuitParams{
			Rank: c.Rank.String(),
			Suit: c.Suit.String(),
		})
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("q.GetCardByRankSuit(): %w", err)
		}
		if _, err := qWithTx.SetCardHandByCardID(ctx, query.SetCardHandByCardIDParams{
			HandID: sql.NullInt64{Int64: hand.ID, Valid: true},
			ID:     dbCard.ID,
		}); err != nil {
			tx.Rollback()
			return fmt.Errorf("q.SetCardHand(): %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx.Commit(): %w", err)
	}

	if err := calcEquity(ctx, query.New(conn)); err != nil {
		return fmt.Errorf("calcEquity: %w", err)
	}

	return nil
}

func MuckPlayer(ctx context.Context, conn *sql.DB, cards []poker.Card) error {
	log.Printf("MuckPlayer: %v", cards)
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("conn.BeginTx(): %w", err)
	}
	defer func() {
		if r := recover(); r != nil || err != nil {
			tx.Rollback()
		}
	}()
	qWithTx := query.New(tx)

	card, err := qWithTx.GetCardByRankSuit(ctx, query.GetCardByRankSuitParams{
		Rank: cards[0].Rank.String(),
		Suit: cards[0].Suit.String(),
	})
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("q.GetCardByRankSuit(): %w", err)
	}
	hand, err := qWithTx.GetHand(ctx, card.HandID.Int64)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("q.GetHandByCardId(): %w", err)
	}

	if err := qWithTx.MuckHand(ctx, hand.ID); err != nil {
		tx.Rollback()
		return fmt.Errorf("q.MuckHand(): %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx.Commit(): %w", err)
	}

	if err := calcEquity(ctx, query.New(conn)); err != nil {
		return fmt.Errorf("calcEquity: %w", err)
	}

	return nil
}
