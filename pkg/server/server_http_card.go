package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
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

func HandleCards(c echo.Context, conn *sql.DB, updatedCh chan struct{}) error {
	defer c.Request().Body.Close()

	input := PostCardRequest{}
	if err := json.NewDecoder(c.Request().Body).Decode(&input); err != nil {
		log.Printf("invalid request body: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	_, err := store.GetAntennaBySerial(c.Request().Context(), conn, input.DeviceID, input.PairID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if err := store.RegisterNewDevice(c.Request().Context(), conn, input.DeviceID, input.PairID); err != nil {
				log.Printf("failed to register new device: %v", err)
				return echo.NewHTTPError(http.StatusInternalServerError, "failed to register new device")
			}
		} else {
			log.Printf("failed to get antenna: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to get antenna")
		}
	}

	uid := strings.ReplaceAll(input.UID, " ", "")

	if err := processCard(c.Request().Context(), conn, config.Conf, uid, input.DeviceID, input.PairID, updatedCh); err != nil {
		log.Printf("failed to process card: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to process card")
	}

	return c.JSON(http.StatusOK, "success to receive card")
}

func processCard(ctx context.Context, conn *sql.DB, cc config.Config, uid string, deviceID string, pairID int, updatedCh chan struct{}) error {
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

			if err := store.AddHand(ctx, conn, []poker.Card{storedCards[0], card}, serial, updatedCh); err != nil {
				return fmt.Errorf("store.AddHand(): %w", err)
			}
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
			if err := store.MuckPlayer(ctx, conn, []poker.Card{storedCards[0], card}, updatedCh); err != nil {
				return fmt.Errorf("store.MuckPlayer(): %w", err)
			}
		}
	case "board":
		// Send anyway if board
		if err := store.AddBoard(ctx, conn, []poker.Card{card}, serial, updatedCh); err != nil {
			if errors.Is(err, store.ErrWillGoToNextGame) {
				// go to next game
				if err := store.ClearGame(ctx, conn); err != nil {
					return fmt.Errorf("store.ClearGame(): %w", err)
				}
				updatedCh <- struct{}{}
				return nil
			}

			return fmt.Errorf("store.AddBoard(): %w", err)
		}
	case "unknown":
		log.Printf("unknown type antenna (serial: %s)", serial)
	}

	return nil
}
