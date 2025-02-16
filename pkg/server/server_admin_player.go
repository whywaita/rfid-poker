package server

import (
	"database/sql"
	"errors"
	"log"
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
	q := query.New(conn)

	players, err := q.GetPlayersWithDevice(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to get players")
	}

	var resp GetAdminPlayersResponse
	for _, p := range players {
		deviceID, pairID, err := store.FromSerial(p.Serial)
		if err != nil {
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
		log.Printf("strconv.Atoi(%s): %v", req.ID, err)
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
	}

	player, err := q.GetPlayer(c.Request().Context(), int32(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return echo.NewHTTPError(http.StatusNotFound, ErrorResponse{Error: "player not found"})
		}
		log.Printf("q.GetPlayer(): %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	if _, err := q.UpdatePlayerName(c.Request().Context(), query.UpdatePlayerNameParams{
		Name: req.Name,
		ID:   player.ID,
	}); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	respPlayer, err := q.GetPlayerWithDevice(c.Request().Context(), player.ID)
	if err != nil {
		log.Printf("q.GetPlayer(%d): %v", id, err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	deviceID, pairID, err := store.FromSerial(respPlayer.Serial)
	if err != nil {
		log.Printf("store.FromSerial(%s): %v", respPlayer.Serial, err)
		return echo.NewHTTPError(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, PostAdminPlayerResponse{
		Player: Player{
			ID:       respPlayer.ID,
			Name:     respPlayer.Name,
			DeviceID: deviceID,
			PairID:   pairID,
		},
	})
}
