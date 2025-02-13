package server

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/whywaita/rfid-poker/pkg/query"

	"github.com/whywaita/poker-go"
	"github.com/whywaita/rfid-poker/pkg/config"
	"github.com/whywaita/rfid-poker/pkg/playercards"

	"github.com/whywaita/rfid-poker/pkg/store"

	"github.com/labstack/echo/v4"
)

type Card struct {
	UID      string `json:"uid"`
	DeviceID string `json:"device_id"`
	PairID   int    `json:"pair_id"`
}

func HandleCards(c echo.Context, conn *sql.DB, updatedCh chan struct{}) error {
	defer c.Request().Body.Close()

	input := Card{}
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

	uid, err := hex.DecodeString(input.UID)
	if err != nil {
		log.Printf("invalid UID: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid UID")
	}

	if err := processCard(c.Request().Context(), conn, config.Conf, string(uid), input.DeviceID, input.PairID, updatedCh); err != nil {
		log.Printf("failed to process card: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to process card")
	}

	return c.JSON(http.StatusOK, "success to receive card")
}

func processCard(ctx context.Context, conn *sql.DB, cc config.Config, uid string, deviceID string, pairID int, updatedCh chan struct{}) error {
	pcard, err := playercards.LoadPlayerCard([]byte(uid), cc.CardIDs)
	if err != nil {
		return fmt.Errorf("playercards.LoadPlayerCard(%s, cardConfigs): %w", hex.EncodeToString([]byte(uid)), err)
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
	var player query.Player
	if strings.EqualFold(antenna.AntennaTypeName, "unknown") {
		player, err = qWithTx.AddPlayer(ctx, fmt.Sprintf("player-%s-%d", deviceID, pairID))
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("query.AddPlayer(): %w", err)
		}
		if err := qWithTx.SetPlayerIDToAntennaBySerial(ctx, query.SetPlayerIDToAntennaBySerialParams{
			PlayerID: sql.NullInt64{Int64: player.ID, Valid: true},
			Serial:   serial,
		}); err != nil {
			tx.Rollback()
			return fmt.Errorf("query.SetPlayerIDToAntennaBySerial(): %w", err)
		}
	} else {
		player, err = qWithTx.GetPlayerBySerial(ctx, store.ToSerial(deviceID, pairID))
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("query.GetPlayerBySerial(): %w", err)
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
		case len(storedCards) == 1 && storedCards[0].Rank != card.Rank && storedCards[0].Suit != card.Suit: // not same card
			if err := store.AddHand(ctx, conn, []poker.Card{storedCards[0], card}, serial); err != nil {
				return fmt.Errorf("store.AddHand(): %w", err)
			}
			updatedCh <- struct{}{}
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
			updatedCh <- struct{}{}
		}
	case "board":
		// Send anyway if board
		if err := store.AddBoard(ctx, conn, []poker.Card{card}); err != nil {
			return fmt.Errorf("store.AddBoard(): %w", err)
		}
		updatedCh <- struct{}{}
	case "unknown":
		log.Println("unknown type antenna")
	}

	return nil
}

// We cached the cards that have been read by the reader.
// key: playerID, value: []poker.Card
var cache sync.Map

func LoadCache(playerID string) []poker.Card {
	loaded, ok := cache.Load(playerID)
	if !ok {
		return nil
	}
	s, ok := loaded.([]poker.Card)
	if !ok {
		return nil
	}
	return s
}

func SaveCache(playerID string, cards poker.Card) {
	cached := LoadCache(playerID)
	if cached == nil {
		cache.Store(playerID, []poker.Card{cards})
		return
	}

	for _, v := range cached {
		if cards == v {
			// already cached
			return
		}
	}

	cache.Store(playerID, append(cached, cards))
}

func ClearCache(playerID string) {
	cache.Delete(playerID)
}
