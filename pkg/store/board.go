package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"sort"

	"github.com/whywaita/poker-go"
	"github.com/whywaita/rfid-poker/pkg/query"
)

var (
	ErrWillGoToNextGame = errors.New("will go to next game")
)

func AddBoard(ctx context.Context, conn *sql.DB, cards []poker.Card, serial string) (bool, error) {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("conn.BeginTx(): %w", err)
	}
	defer func() {
		if r := recover(); r != nil || err != nil {
			tx.Rollback()
		}
	}()
	qWithTx := query.New(tx)

	nowBoard, err := GetBoardAll(ctx, qWithTx)
	if err != nil {
		tx.Rollback()
		return false, fmt.Errorf("GetBoard(): %w", err)
	}

	board, needInsert, isUpdated := concatCards(nowBoard, cards)
	if len(board) >= 6 {
		// load 7 cards. will go to next game.
		tx.Rollback()
		return false, ErrWillGoToNextGame
	}

	if len(needInsert) > 0 {
		for _, c := range needInsert {
			err := qWithTx.AddCardToBoard(ctx, query.AddCardToBoardParams{
				CardSuit: c.Suit.String(),
				CardRank: c.Rank.String(),
				Serial:   serial,
			})
			if err != nil {
				tx.Rollback()
				return false, fmt.Errorf("query.AddCardToBoard(): %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("tx.Commit(): %w", err)
	}

	return isUpdated, nil
}

func GetBoardAll(ctx context.Context, q *query.Queries) ([]poker.Card, error) {
	cards, err := q.GetBoard(ctx)
	if err != nil {
		return nil, fmt.Errorf("db.GetBoard(): %w", err)
	}

	var board []poker.Card
	for _, c := range cards {
		card, err := query.Card{CardSuit: c.CardSuit, CardRank: c.CardRank}.ToPokerGo()
		if err != nil {
			return nil, fmt.Errorf("card.ToPokerGo(): %w", err)
		}
		board = append(board, *card)
	}

	return board, nil
}

func GetBoard(ctx context.Context, q *query.Queries) ([]poker.Card, error) {
	cards, err := q.GetBoard(ctx)
	if err != nil {
		return nil, fmt.Errorf("db.GetBoard(): %w", err)
	}

	// use only 5 cards order by oldest
	sort.SliceStable(cards, func(i, j int) bool {
		return cards[i].ID < cards[j].ID
	})
	if len(cards) > 5 {
		cards = cards[:5]
	}

	var board []poker.Card
	for _, c := range cards {
		card, err := query.Card{CardSuit: c.CardSuit, CardRank: c.CardRank}.ToPokerGo()
		if err != nil {
			return nil, fmt.Errorf("card.ToPokerGo(): %w", err)
		}
		board = append(board, *card)
	}

	return board, nil
}

// concatCards concat already stored cards and new cards (remove duplicated)
func concatCards(already, newCards []poker.Card) ([]poker.Card, []poker.Card, bool) {
	concat := make([]poker.Card, 0)
	concat = append(concat, already...)
	needInsert := make([]poker.Card, 0)

	isUpdated := false

	for _, newCard := range newCards {
		if slices.Contains(concat, newCard) {
			// already stored, ignore
			continue
		}
		concat = append(concat, newCard)
		needInsert = append(needInsert, newCard)
		isUpdated = true
	}

	return concat, needInsert, isUpdated
}
