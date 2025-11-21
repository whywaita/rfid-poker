package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sort"

	"github.com/whywaita/poker-go"
	"github.com/whywaita/rfid-poker/pkg/query"
)

var (
	ErrBoardCardLimitExceeded = errors.New("board card limit exceeded (max 5 cards)")
)

func AddBoard(ctx context.Context, conn *sql.DB, cards []poker.Card, serial string) (bool, error) {
	// Get or create current game
	gameID, err := GetOrCreateCurrentGame(ctx, conn)
	if err != nil {
		return false, fmt.Errorf("GetOrCreateCurrentGame(): %w", err)
	}

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

	// Check if adding new cards would exceed the limit (max 5 board cards)
	if len(board) > 5 {
		tx.Rollback()
		slog.WarnContext(ctx, "Board card limit exceeded, rejecting request",
			slog.String("game_id", gameID),
			slog.String("event", "board_card_limit_exceeded"),
			slog.Int("current_board_count", len(nowBoard)),
			slog.Int("attempted_total", len(board)))
		return false, ErrBoardCardLimitExceeded
	}

	if len(needInsert) > 0 {
		for _, c := range needInsert {
			err := qWithTx.AddCardToBoard(ctx, query.AddCardToBoardParams{
				CardSuit: c.Suit.String(),
				CardRank: c.Rank.String(),
				Serial:   serial,
				GameID:   gameID,
			})
			if err != nil {
				tx.Rollback()
				return false, fmt.Errorf("query.AddCardToBoard(): %w", err)
			}
			slog.InfoContext(ctx, "Added board card",
				slog.String("game_id", gameID),
				slog.String("event", "board_card_added"),
				slog.String("card_rank", c.Rank.String()),
				slog.String("card_suit", c.Suit.String()),
				slog.Bool("is_board", true),
				slog.String("serial", serial),
				slog.Int("board_card_count", len(board)+1))
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
