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

	_, err := store.GetAntennaBySerial(c.Request().Context(), conn, input.DeviceID, input.PairID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if err := store.RegisterNewDevice(c.Request().Context(), conn, input.DeviceID, input.PairID); err != nil {
				logger.WarnContext(c.Request().Context(), "failed to register new device", "error", err)
				return echo.NewHTTPError(http.StatusInternalServerError, "failed to register new device")
			}
		} else {
			logger.WarnContext(c.Request().Context(), "failed to get antenna", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to get antenna")
		}
	}

	if err := processCard(c.Request().Context(), conn, config.Conf, uid, input.DeviceID, input.PairID); err != nil {
		logger.WarnContext(c.Request().Context(), "failed to process card", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to process card")
	}

	return c.JSON(http.StatusOK, "success to receive card")
}

func processCard(ctx context.Context, conn *sql.DB, cc config.Config, uid string, deviceID string, pairID int) error {
	logger := slog.With("method", "processCard")
	pcard, err := playercards.LoadPlayerCard(uid, cc.CardIDs)
	if err != nil {
		return fmt.Errorf("playercards.LoadPlayerCard(%s, cardConfigs): %w", uid, err)
	}
	card, err := playercards.UnmarshalPlayerCard(pcard)
	if err != nil {
		return fmt.Errorf("playercards.UnmarshalPlayerCard(%s): %w", pcard, err)
	}

	serial := store.ToSerial(deviceID, pairID)

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

	antenna, err := qWithTx.GetAntennaBySerial(ctx, serial)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("query.GetAntennaBySerial(): %w", err)
	}

	// if unknown, register new player
	if strings.EqualFold(antenna.AntennaTypeName, "unknown") {
		resultPlayer, err := qWithTx.AddPlayer(ctx, fmt.Sprintf("player-%s-%d", deviceID, pairID))
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("query.AddPlayer(): %w", err)
		}
		playerID, err := resultPlayer.LastInsertId()
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("resultPlayer.LastInsertId(): %w", err)
		}
		if err := qWithTx.SetPlayerIDToAntennaBySerial(ctx, query.SetPlayerIDToAntennaBySerialParams{
			PlayerID: sql.NullInt32{Int32: int32(playerID), Valid: true},
			Serial:   serial,
		}); err != nil {
			tx.Rollback()
			return fmt.Errorf("query.SetPlayerIDToAntennaBySerial(): %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx.Commit(): %w", err)
	}

	newAntenna, err := store.GetAntennaBySerial(ctx, conn, deviceID, pairID)
	if err != nil {
		return fmt.Errorf("query.GetAntennaBySerial(): %w", err)
	}

	switch newAntenna.AntennaTypeName {
	case "player":
		storedCards, err := store.GetCardBySerial(ctx, conn, serial)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("store.GetCardBySerial(): %w", err)
		}

		switch {
		case len(storedCards) == 0:
			if err := store.AddCard(ctx, conn, card, serial); err != nil {
				return fmt.Errorf("store.AddCard(): %w", err)
			}
		case len(storedCards) == 1:
			// if same card, do nothing
			if storedCards[0].Rank == card.Rank && storedCards[0].Suit == card.Suit {
				return nil
			}

			if err := store.AddHand(ctx, conn, []poker.Card{storedCards[0], card}, serial); err != nil {
				return fmt.Errorf("store.AddHand(): %w", err)
			}
			notifyClients()

			go func() {
				if err := store.CalcEquity(context.Background(), query.New(conn)); err != nil {
					logger.WarnContext(context.Background(), "calcEquity", "error", err)
				}
				notifyClients()
			}()
		}
	case "muck":
		storedCards, err := store.GetCardBySerial(ctx, conn, serial)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("store.GetCardBySerial(): %w", err)
		}
		switch {
		case len(storedCards) == 0:
			if err := store.AddCard(ctx, conn, card, serial); err != nil {
				return fmt.Errorf("store.AddCard(): %w", err)
			}
		case len(storedCards) == 1 && storedCards[0].Rank != card.Rank && storedCards[0].Suit != card.Suit: // not same card
			if err := store.MuckPlayer(ctx, conn, []poker.Card{storedCards[0], card}); err != nil {
				return fmt.Errorf("store.MuckPlayer(): %w", err)
			}
			notifyClients()

			go func() {
				if err := store.CalcEquity(context.Background(), query.New(conn)); err != nil {
					logger.WarnContext(context.Background(), "calcEquity", "error", err)
				}
				notifyClients()
			}()
		}
	case "board":
		// Send anyway if board
		isUpdated, err := store.AddBoard(ctx, conn, []poker.Card{card}, serial)
		if err != nil {
			if errors.Is(err, store.ErrWillGoToNextGame) {
				// go to next game
				if err := store.ClearGame(ctx, conn); err != nil {
					return fmt.Errorf("store.ClearGame(): %w", err)
				}
				notifyClients()
				return nil
			}

			return fmt.Errorf("store.AddBoard(): %w", err)
		}
		notifyClients()

		go func(isUpdated bool) {
			if isUpdated {
				if err := store.CalcEquity(context.Background(), query.New(conn)); err != nil {
					logger.WarnContext(context.Background(), "calcEquity", "error", err)
				}
				notifyClients()
			}
		}(isUpdated)
	case "unknown":
		logger.WarnContext(ctx, "unknown type antenna", "serial", serial)
	}

	return nil
}
