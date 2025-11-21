package store

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"

	"github.com/google/uuid"
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

// CreateNewGame creates a new game with a UUID and returns the game ID
func CreateNewGame(ctx context.Context, conn *sql.DB) (string, error) {
	db := query.New(conn)
	gameID := uuid.New().String()

	if err := db.CreateGame(ctx, gameID); err != nil {
		return "", fmt.Errorf("db.CreateGame(): %w", err)
	}

	slog.InfoContext(ctx, "New game started",
		slog.String("game_id", gameID),
		slog.String("event", "game_started"))
	return gameID, nil
}

// GetOrCreateCurrentGame returns the current active game ID, or creates a new one if none exists
// This function uses a transaction to ensure atomicity of the check-and-create operation
func GetOrCreateCurrentGame(ctx context.Context, conn *sql.DB) (string, error) {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("conn.BeginTx(): %w", err)
	}
	defer func() {
		if r := recover(); r != nil || err != nil {
			tx.Rollback()
		}
	}()

	db := query.New(tx)

	game, err := db.GetCurrentGame(ctx)
	if err == nil {
		// Game exists, commit and return
		if err := tx.Commit(); err != nil {
			return "", fmt.Errorf("tx.Commit(): %w", err)
		}
		return game.ID, nil
	}

	if err != sql.ErrNoRows {
		tx.Rollback()
		return "", fmt.Errorf("db.GetCurrentGame(): %w", err)
	}

	// No active game, create a new one within the same transaction
	gameID := uuid.New().String()
	if err := db.CreateGame(ctx, gameID); err != nil {
		tx.Rollback()
		return "", fmt.Errorf("db.CreateGame(): %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("tx.Commit(): %w", err)
	}

	slog.InfoContext(ctx, "New game started",
		slog.String("game_id", gameID),
		slog.String("event", "game_started"))

	return gameID, nil
}

// FinishCurrentGame finishes the current active game
func FinishCurrentGame(ctx context.Context, conn *sql.DB) error {
	db := query.New(conn)

	game, err := db.GetCurrentGame(ctx)
	if err == sql.ErrNoRows {
		// No active game to finish
		return nil
	}
	if err != nil {
		return fmt.Errorf("db.GetCurrentGame(): %w", err)
	}

	if err := db.FinishGame(ctx, game.ID); err != nil {
		return fmt.Errorf("db.FinishGame(): %w", err)
	}

	slog.InfoContext(ctx, "Game finished",
		slog.String("game_id", game.ID),
		slog.String("event", "game_finished"),
		slog.Time("started_at", game.StartedAt))
	return nil
}

func ClearGame(ctx context.Context, conn *sql.DB) error {
	db := query.New(conn)

	// Get current game before finishing
	game, err := db.GetCurrentGame(ctx)
	if err == sql.ErrNoRows {
		// No active game to clear
		return nil
	}
	if err != nil {
		return fmt.Errorf("db.GetCurrentGame(): %w", err)
	}

	gameID := game.ID

	// Archive hands to hand_history before clearing
	if err := db.CopyHandsToHistory(ctx, gameID); err != nil {
		return fmt.Errorf("db.CopyHandsToHistory(): %w", err)
	}

	// Finish current game
	if err := db.FinishGame(ctx, gameID); err != nil {
		return fmt.Errorf("db.FinishGame(): %w", err)
	}

	// Delete cards and hands for this game
	if err := db.DeleteCardByGameID(ctx, gameID); err != nil {
		return fmt.Errorf("db.DeleteCardByGameID(): %w", err)
	}

	if err := db.DeleteHandByGameID(ctx, gameID); err != nil {
		return fmt.Errorf("db.DeleteHandByGameID(): %w", err)
	}

	// Delete the game itself
	if err := db.DeleteGameByID(ctx, gameID); err != nil {
		return fmt.Errorf("db.DeleteGameByID(): %w", err)
	}

	slog.InfoContext(ctx, "Game cleared and archived",
		slog.String("game_id", gameID),
		slog.String("event", "game_cleared"),
		slog.Time("started_at", game.StartedAt))

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
