package server

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/whywaita/rfid-poker/pkg/store"

	"github.com/labstack/echo/v4"
	"github.com/whywaita/rfid-poker/pkg/query"
)

type Player struct {
	ID   int32  `json:"id"`
	Name string `json:"name"`

	DeviceID string `json:"device_id"`
	PairID   int    `json:"pair_id"`
}

type GetAdminPlayersResponse struct {
	Players []Player `json:"players"`
}

func HandleGetAdminPlayers(c echo.Context, conn *sql.DB) error {
	logger := slog.With("method", "HandleGetAdminPlayers")
	q := query.New(conn)

	players, err := q.GetPlayersWithDevice(c.Request().Context())
	if err != nil {
		logger.WarnContext(c.Request().Context(), "q.GetPlayersWithDevice", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get players")
	}

	var resp GetAdminPlayersResponse
	for _, p := range players {
		logger = logger.With("player_id", p.ID)
		deviceID, pairID, err := store.FromSerial(p.Serial)
		if err != nil {
			logger.WarnContext(c.Request().Context(), "store.FromSerial", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to convert serial")
		}
		resp.Players = append(resp.Players, Player{
			ID:       p.ID,
			Name:     p.Name,
			DeviceID: deviceID,
			PairID:   pairID,
		})
	}

	return c.JSON(http.StatusOK, resp)
}

type PostAdminPlayerRequest struct {
	ID   string `param:"id"`
	Name string `json:"name"`
}

type PostAdminPlayerResponse struct {
	Player Player `json:"player"`
}

func HandlePostAdminPlayer(c echo.Context, conn *sql.DB) error {
	logger := slog.With("method", "HandlePostAdminPlayer")
	q := query.New(conn)

	var req PostAdminPlayerRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}

	id, err := strconv.Atoi(req.ID)
	if err != nil {
		logger.WarnContext(c.Request().Context(), "strconv.Atoi", "error", err, slog.String("id", req.ID))
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
	}

	tx, err := conn.BeginTx(c.Request().Context(), nil)
	if err != nil {
		logger.WarnContext(c.Request().Context(), "conn.BeginTx", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}
	defer tx.Rollback()

	qWithTx := q.WithTx(tx)

	player, err := qWithTx.GetPlayer(c.Request().Context(), int32(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, ErrorResponse{Error: "player not found"})
		}
		logger.WarnContext(c.Request().Context(), "qWithTx.GetPlayer", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	if _, err := qWithTx.UpdatePlayerName(c.Request().Context(), query.UpdatePlayerNameParams{
		Name: req.Name,
		ID:   player.ID,
	}); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	respPlayer, err := qWithTx.GetPlayerWithDevice(c.Request().Context(), player.ID)
	if err != nil {
		logger.WarnContext(c.Request().Context(), "qWithTx.GetPlayerWithDevice", "error", err, slog.Int64("player_id", int64(player.ID)))
		return echo.NewHTTPError(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	deviceID, pairID, err := store.FromSerial(respPlayer.Serial)
	if err != nil {
		logger.WarnContext(c.Request().Context(), "store.FromSerial", "error", err, slog.String("serial", respPlayer.Serial))
		return echo.NewHTTPError(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	if err := tx.Commit(); err != nil {
		logger.WarnContext(c.Request().Context(), "tx.Commit", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	notifyClients()

	return c.JSON(http.StatusOK, PostAdminPlayerResponse{
		Player: Player{
			ID:       respPlayer.ID,
			Name:     respPlayer.Name,
			DeviceID: deviceID,
			PairID:   pairID,
		},
	})
}
