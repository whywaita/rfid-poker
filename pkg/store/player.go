package store

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sort"

	"entgo.io/ent/dialect/sql/sqlgraph"
	"github.com/whywaita/poker-go"
	"github.com/whywaita/rfid-poker/pkg/query"
)

func AddHand(ctx context.Context, conn *sql.DB, input []poker.Card, serial string, updatedCh chan struct{}) error {
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
	handResult, err := hand.LastInsertId()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("hand.LastInsertId(): %w", err)
	}

	for _, c := range input {
		_, err = qWithTx.AddCard(ctx, query.AddCardParams{
			CardSuit: c.Suit.String(),
			CardRank: c.Rank.String(),
			Serial:   serial,
			IsBoard:  false,
		})
		if err != nil && !sqlgraph.IsUniqueConstraintError(err) {
			tx.Rollback()
			return fmt.Errorf("q.AddCard(): %w", err)
		}

		dbCard, err := qWithTx.GetCardByRankSuit(ctx, query.GetCardByRankSuitParams{
			CardRank: c.Rank.String(),
			CardSuit: c.Suit.String(),
		})
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("q.GetCardByRankSuit(): %w", err)
		}
		if _, err := qWithTx.SetCardHandByCardID(ctx, query.SetCardHandByCardIDParams{
			HandID: sql.NullInt32{Int32: int32(handResult), Valid: true},
			ID:     dbCard.ID,
		}); err != nil {
			tx.Rollback()
			return fmt.Errorf("q.SetCardHand(): %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx.Commit(): %w", err)
	}

	updatedCh <- struct{}{}

	go func() {
		if err := calcEquity(context.Background(), query.New(conn), updatedCh); err != nil {
			log.Printf("calcEquity: %v", err)
		}
	}()
	return nil
}

func MuckPlayer(ctx context.Context, conn *sql.DB, cards []poker.Card, updatedCh chan struct{}) error {
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
		CardRank: cards[0].Rank.String(),
		CardSuit: cards[0].Suit.String(),
	})
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("q.GetCardByRankSuit(): %w", err)
	}
	hand, err := qWithTx.GetHand(ctx, card.HandID.Int32)
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

	updatedCh <- struct{}{}

	go func() {
		if err := calcEquity(context.Background(), query.New(conn), updatedCh); err != nil {
			log.Printf("calcEquity: %v", err)
		}
	}()

	return nil
}
