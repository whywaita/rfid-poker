package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/whywaita/rfid-poker/pkg/store"

	"github.com/labstack/echo/v4"
	"github.com/whywaita/poker-go"

	"github.com/whywaita/rfid-poker/pkg/query"
)

type Card struct {
	Suit string `json:"suit"`
	Rank string `json:"rank"`
}

type Hand struct {
	ID       int32   `json:"id"`
	PlayerID int32   `json:"player_id"`
	IsMuck   bool    `json:"is_muck"`
	Card     [2]Card `json:"cards"`
}

type GetAdminPlayerHandResponse struct {
	Hand Hand `json:"hand"`
}

func HandleGetAdminPlayerHand(c echo.Context, conn *sql.DB) error {
	logger := slog.With("method", "HandleGetAdminPlayerHand")
	q := query.New(conn)

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.WarnContext(c.Request().Context(), "strconv.Atoi", "error", err, slog.String("id", c.Param("id")))
		return echo.NewHTTPError(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
	}

	storedHand, err := q.GetHandWithCardByPlayerID(c.Request().Context(), int32(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, ErrorResponse{Error: fmt.Sprintf("not found: (player_id: %d)", id)})
		}

		logger.WarnContext(c.Request().Context(), "q.GetHandWithCardByPlayerID", "error", err, slog.Int("player_id", id))
		return echo.NewHTTPError(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	cardASuit := poker.UnmarshalSuitString(storedHand.CardASuit)
	cardARank := poker.UnmarshalRankString(storedHand.CardARank)
	cardBSuit := poker.UnmarshalSuitString(storedHand.CardBSuit)
	cardBRank := poker.UnmarshalRankString(storedHand.CardBRank)

	// check unknown card
	if cardASuit == -1 || cardARank == poker.RankUnknown || cardBSuit == -1 || cardBRank == poker.RankUnknown {
		logger.WarnContext(c.Request().Context(), "found invalid card", "hand", storedHand)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	resp := GetAdminPlayerHandResponse{
		Hand: Hand{
			ID:       storedHand.HandID,
			PlayerID: storedHand.PlayerID,
			IsMuck:   storedHand.IsMuck,
			Card: [2]Card{
				{
					Suit: cardASuit.String(),
					Rank: cardARank.String(),
				},
				{
					Suit: cardBSuit.String(),
					Rank: cardBRank.String(),
				},
			},
		},
	}

	return c.JSON(http.StatusOK, resp)
}

func HandleDeleteAdminPlayerHand(c echo.Context, conn *sql.DB) error {
	logger := slog.With("method", "HandleDeleteAdminPlayerHand")
	q := query.New(conn)

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.WarnContext(c.Request().Context(), "strconv.Atoi", "error", err, slog.String("id", c.Param("id")))
		return echo.NewHTTPError(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
	}

	hand, err := q.GetHandWithCardByPlayerID(c.Request().Context(), int32(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, ErrorResponse{Error: fmt.Sprintf("not found: (player_id: %d)", id)})
		}
		logger.WarnContext(c.Request().Context(), "q.GetHandWithCardByPlayerID", "error", err, slog.Int("player_id", id))
		return echo.NewHTTPError(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	if err := store.MuckPlayer(c.Request().Context(), conn, []poker.Card{
		{
			Suit: poker.UnmarshalSuitString(hand.CardASuit),
			Rank: poker.UnmarshalRankString(hand.CardARank),
		},
		{
			Suit: poker.UnmarshalSuitString(hand.CardBSuit),
			Rank: poker.UnmarshalRankString(hand.CardBRank),
		},
	}); err != nil {
		logger.WarnContext(c.Request().Context(), "store.MuckPlayer", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	notifyClients()

	go func() {
		if err := store.CalcEquity(context.Background(), query.New(conn)); err != nil {
			logger.WarnContext(c.Request().Context(), "store.CalcEquity", "error", err)
		}
		notifyClients()
	}()

	return c.JSON(http.StatusNoContent, nil)
}
