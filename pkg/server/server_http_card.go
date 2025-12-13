package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/whywaita/poker-go"
	"github.com/whywaita/rfid-poker/pkg/config"
	"github.com/whywaita/rfid-poker/pkg/playercards"
	"github.com/whywaita/rfid-poker/pkg/query"
	"github.com/whywaita/rfid-poker/pkg/store"

	"github.com/labstack/echo/v4"
)

type PostCardRequest struct {
	UID      string `json:"uid"`
	DeviceID string `json:"device_id"`
	PairID   int    `json:"pair_id"`
}

// serialLocks manages per-serial mutex locks to prevent race conditions
// when processing cards from the same device
var serialLocks = struct {
	sync.Mutex
	m map[string]*sync.Mutex
}{m: make(map[string]*sync.Mutex)}

// getSerialLock returns a mutex for the given serial, creating one if it doesn't exist
func getSerialLock(serial string) *sync.Mutex {
	serialLocks.Lock()
	defer serialLocks.Unlock()

	if serialLocks.m[serial] == nil {
		serialLocks.m[serial] = &sync.Mutex{}
	}
	return serialLocks.m[serial]
}

func HandleCards(c echo.Context, conn *sql.DB) error {
	logger := slog.With("method", "HandleCards")
	defer c.Request().Body.Close()

	input := PostCardRequest{}
	if err := json.NewDecoder(c.Request().Body).Decode(&input); err != nil {
		logger.WarnContext(c.Request().Context(), "invalid request body", "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	uid := strings.ReplaceAll(input.UID, " ", "")
	logger = logger.With("device_id", input.DeviceID, "pair_id", input.PairID, "uid", input.UID)

	// First, check if this device_id corresponds to a board antenna
	// Board antennas should be treated as one board regardless of pair_id
	boardAntenna, boardErr := store.GetBoardAntennaByDeviceID(c.Request().Context(), conn, input.DeviceID)
	if boardErr == nil {
		// This is a board device, use the existing board antenna
		// Don't register as a new device, just proceed with processing
		logger.InfoContext(c.Request().Context(), "using existing board antenna", "serial", boardAntenna.Serial)
	} else if errors.Is(boardErr, sql.ErrNoRows) {
		// Not a board device, check if antenna exists with device_id-pair_id
		_, err := store.GetAntennaBySerial(c.Request().Context(), conn, input.DeviceID, input.PairID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// Register as new device
				if err := store.RegisterNewDevice(c.Request().Context(), conn, input.DeviceID, input.PairID); err != nil {
					logger.WarnContext(c.Request().Context(), "failed to register new device", "error", err)
					return echo.NewHTTPError(http.StatusInternalServerError, "failed to register new device")
				}
			} else {
				logger.WarnContext(c.Request().Context(), "failed to get antenna", "error", err)
				return echo.NewHTTPError(http.StatusInternalServerError, "failed to get antenna")
			}
		}
	} else {
		logger.WarnContext(c.Request().Context(), "failed to check board antenna", "error", boardErr)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to check board antenna")
	}

	modified, err := processCard(c.Request().Context(), conn, config.Conf, uid, input.DeviceID, input.PairID)
	if err != nil {
		logger.WarnContext(c.Request().Context(), "failed to process card", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to process card")
	}

	if !modified {
		return c.NoContent(http.StatusNotModified)
	}

	return c.JSON(http.StatusOK, "success to receive card")
}

// processCard processes a card and returns (modified, error).
// modified is true if the card was newly added or updated, false if it already existed.
func processCard(ctx context.Context, conn *sql.DB, cc config.Config, uid string, deviceID string, pairID int) (bool, error) {
	logger := slog.With("method", "processCard")
	pcard, err := playercards.LoadPlayerCard(uid, cc.CardIDs)
	if err != nil {
		return false, fmt.Errorf("playercards.LoadPlayerCard(%s, cardConfigs): %w", uid, err)
	}
	card, err := playercards.UnmarshalPlayerCard(pcard)
	if err != nil {
		return false, fmt.Errorf("playercards.UnmarshalPlayerCard(%s): %w", pcard, err)
	}

	// Check if this device_id corresponds to a board antenna
	// If so, use the board antenna's serial instead of device_id-pair_id
	var serial string
	boardAntenna, boardErr := store.GetBoardAntennaByDeviceID(ctx, conn, deviceID)
	if boardErr == nil {
		// This is a board device, use the board antenna's serial
		serial = boardAntenna.Serial
		logger.InfoContext(ctx, "using board antenna serial", "serial", serial)
	} else {
		// Not a board device, use device_id-pair_id as serial
		serial = store.ToSerial(deviceID, pairID)
	}

	// Lock per serial to prevent race conditions when multiple cards arrive simultaneously
	serialMu := getSerialLock(serial)
	serialMu.Lock()
	defer serialMu.Unlock()

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

	antenna, err := qWithTx.GetAntennaBySerial(ctx, serial)
	if err != nil {
		tx.Rollback()
		return false, fmt.Errorf("query.GetAntennaBySerial(): %w", err)
	}

	// if unknown, register new player
	if strings.EqualFold(antenna.AntennaTypeName, "unknown") {
		resultPlayer, err := qWithTx.AddPlayer(ctx, fmt.Sprintf("player-%s-%d", deviceID, pairID))
		if err != nil {
			tx.Rollback()
			return false, fmt.Errorf("query.AddPlayer(): %w", err)
		}
		playerID, err := resultPlayer.LastInsertId()
		if err != nil {
			tx.Rollback()
			return false, fmt.Errorf("resultPlayer.LastInsertId(): %w", err)
		}
		if err := qWithTx.SetPlayerIDToAntennaBySerial(ctx, query.SetPlayerIDToAntennaBySerialParams{
			PlayerID: sql.NullInt32{Int32: int32(playerID), Valid: true},
			Serial:   serial,
		}); err != nil {
			tx.Rollback()
			return false, fmt.Errorf("query.SetPlayerIDToAntennaBySerial(): %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("tx.Commit(): %w", err)
	}

	// Get the antenna again using the same logic as above
	var newAntenna *query.GetAntennaBySerialRow
	if boardErr == nil {
		// Board antenna - get directly by serial
		q := query.New(conn)
		antenna, err := q.GetAntennaBySerial(ctx, serial)
		if err != nil {
			return false, fmt.Errorf("query.GetAntennaBySerial(): %w", err)
		}
		newAntenna = &antenna
	} else {
		// Regular antenna - use the helper function
		newAntenna, err = store.GetAntennaBySerial(ctx, conn, deviceID, pairID)
		if err != nil {
			return false, fmt.Errorf("store.GetAntennaBySerial(): %w", err)
		}
	}

	switch newAntenna.AntennaTypeName {
	case "player":
		storedCards, err := store.GetCardBySerial(ctx, conn, serial)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return false, fmt.Errorf("store.GetCardBySerial(): %w", err)
		}

		switch {
		case len(storedCards) == 0:
			if err := store.AddCard(ctx, conn, card, serial); err != nil {
				return false, fmt.Errorf("store.AddCard(): %w", err)
			}
		case len(storedCards) == 1:
			// if same card, do nothing
			if storedCards[0].Rank == card.Rank && storedCards[0].Suit == card.Suit {
				return false, nil // Card already exists, not modified
			}

			if err := store.AddHand(ctx, conn, []poker.Card{storedCards[0], card}, serial); err != nil {
				return false, fmt.Errorf("store.AddHand(): %w", err)
			}
			notifyClients()

			go func() {
				if err := store.CalcEquity(context.Background(), query.New(conn)); err != nil {
					logger.WarnContext(context.Background(), "calcEquity", "error", err)
				}
				notifyClients()
			}()
		default:
			// len(storedCards) >= 2, hand already complete
			return false, nil
		}
	case "muck":
		storedCards, err := store.GetCardBySerial(ctx, conn, serial)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return false, fmt.Errorf("store.GetCardBySerial(): %w", err)
		}
		switch {
		case len(storedCards) == 0:
			if err := store.AddCard(ctx, conn, card, serial); err != nil {
				return false, fmt.Errorf("store.AddCard(): %w", err)
			}
		case len(storedCards) == 1 && storedCards[0].Rank != card.Rank && storedCards[0].Suit != card.Suit: // not same card
			if err := store.MuckPlayer(ctx, conn, []poker.Card{storedCards[0], card}); err != nil {
				return false, fmt.Errorf("store.MuckPlayer(): %w", err)
			}
			notifyClients()

			go func() {
				if err := store.CalcEquity(context.Background(), query.New(conn)); err != nil {
					logger.WarnContext(context.Background(), "calcEquity", "error", err)
				}
				notifyClients()
			}()
		default:
			// Same card or already processed
			return false, nil
		}
	case "board":
		// Send anyway if board
		isUpdated, err := store.AddBoard(ctx, conn, []poker.Card{card}, serial)
		if err != nil {
			if errors.Is(err, store.ErrBoardCardLimitExceeded) {
				// Board card limit exceeded, reject the request without saving
				logger.WarnContext(ctx, "board card limit exceeded, rejecting card",
					"serial", serial,
					"card", fmt.Sprintf("%s%s", card.Rank.String(), card.Suit.String()))
				return false, nil // Don't return error to avoid 500, just ignore the card
			}
			return false, fmt.Errorf("store.AddBoard(): %w", err)
		}
		if !isUpdated {
			return false, nil // Card already exists on board
		}
		notifyClients()

		go func() {
			if err := store.CalcEquity(context.Background(), query.New(conn)); err != nil {
				logger.WarnContext(context.Background(), "calcEquity", "error", err)
			}
			notifyClients()
		}()
	case "unknown":
		logger.WarnContext(ctx, "unknown type antenna", "serial", serial)
		return false, nil
	}

	// Update the last card read time for timeout detection
	updateLastCardReadTime(serial, newAntenna.AntennaTypeName)

	return true, nil
}
