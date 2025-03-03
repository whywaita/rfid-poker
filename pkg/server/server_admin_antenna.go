package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/whywaita/rfid-poker/pkg/query"
	"github.com/whywaita/rfid-poker/pkg/store"
)

type Antenna struct {
	ID              int32  `json:"id"`
	DeviceID        string `json:"device_id"`
	PairID          int    `json:"pair_id"`
	AntennaTypeName string `json:"antenna_type_name"`
}

type GetAdminAntennaResponse struct {
	Antenna []Antenna `json:"antenna"`
}

func HandleGetAdminAntenna(c echo.Context, conn *sql.DB) error {
	logger := slog.With("method", "HandleGetAdminAntenna")
	q := query.New(conn)

	antenna, err := q.GetAntenna(c.Request().Context())
	if err != nil {
		logger.WarnContext(c.Request().Context(), "q.GetAntenna", "error", err)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	var respAntenna []Antenna
	for _, a := range antenna {
		deviceID, pairID, err := store.FromSerial(a.Serial)
		if err != nil {
			logger.WarnContext(c.Request().Context(), "store.FromSerial", "error", err)
			return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
		}
		respAntenna = append(respAntenna, Antenna{
			ID:              a.ID,
			DeviceID:        deviceID,
			PairID:          pairID,
			AntennaTypeName: a.AntennaTypeName,
		})
	}

	resp := GetAdminAntennaResponse{
		Antenna: respAntenna,
	}

	return c.JSON(http.StatusOK, resp)
}

type PostAdminAntennaRequest struct {
	ID              string `param:"id" json:"id"`
	AntennaTypeName string `json:"antenna_type_name"`
}

func HandlePostAdminAntenna(c echo.Context, conn *sql.DB) error {
	logger := slog.With("method", "HandlePostAdminAntenna")
	q := query.New(conn)

	var req PostAdminAntennaRequest
	if err := c.Bind(&req); err != nil {
		logger.WarnContext(c.Request().Context(), "c.Bind", "error", err)
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
	}

	id, err := strconv.Atoi(req.ID)
	if err != nil {
		logger.WarnContext(c.Request().Context(), "strconv.Atoi", "error", err, slog.String("id", req.ID))
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
	}

	// check antenna type name in request is valid
	if store.GetAntennaType(req.AntennaTypeName) == store.AntennaTypeUnknown {
		logger.WarnContext(c.Request().Context(), "antenna type name is unknown", slog.String("antenna_type_name", req.AntennaTypeName))
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: fmt.Sprintf("antenna type name (input: %s) is unknown", req.AntennaTypeName)})
	}

	storedAntennas, err := q.GetAntenna(c.Request().Context())
	if err != nil {
		logger.WarnContext(c.Request().Context(), "q.GetAntenna", "error", err)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	// board antenna and muck antenna is only one
	if req.AntennaTypeName == store.AntennaTypeBoard.String() || req.AntennaTypeName == store.AntennaTypeMuck.String() {
		for _, a := range storedAntennas {
			if a.AntennaTypeName == req.AntennaTypeName {
				logger.WarnContext(c.Request().Context(), "antenna type name is already exists", slog.String("antenna_type_name", req.AntennaTypeName))
				return c.JSON(http.StatusBadRequest, ErrorResponse{Error: fmt.Sprintf("antenna type name %s is already exists", req.AntennaTypeName)})
			}
		}
	}

	antenna, err := q.GetAntennaById(c.Request().Context(), int32(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		}
		logger.WarnContext(c.Request().Context(), "q.GetAntennaById", "error", err, slog.Int("id", id))
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	if _, err := q.GetAntennaTypeIdByAntennaTypeName(c.Request().Context(), req.AntennaTypeName); err != nil {
		logger.WarnContext(c.Request().Context(), "q.GetAntennaTypeIdByAntennaTypeName", "error", err)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	if _, err := q.SetAntennaTypeToAntennaBySerial(c.Request().Context(), query.SetAntennaTypeToAntennaBySerialParams{
		Name:   req.AntennaTypeName,
		Serial: antenna.Serial,
	}); err != nil {
		logger.WarnContext(c.Request().Context(), "q.SetAntennaTypeToAntennaBySerial", "error", err)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	if err := cleansingObjectWithChangeAntennaType(
		c.Request().Context(), q, antenna.ID,
		store.GetAntennaType(antenna.AntennaTypeName),
		store.GetAntennaType(req.AntennaTypeName),
	); err != nil {
		logger.WarnContext(c.Request().Context(), "cleansingObjectWithChangeAntennaType", "error", err)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	notifyClients()

	respAntenna, err := q.GetAntennaById(c.Request().Context(), int32(id))
	if err != nil {
		logger.WarnContext(c.Request().Context(), "q.GetAntennaById", "error", err, slog.Int("id", id))
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	deviceID, pairID, err := store.FromSerial(respAntenna.Serial)
	if err != nil {
		logger.WarnContext(c.Request().Context(), "store.FromSerial", "error", err)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}
	resp := Antenna{
		ID:              respAntenna.ID,
		DeviceID:        deviceID,
		PairID:          pairID,
		AntennaTypeName: respAntenna.AntennaTypeName,
	}

	return c.JSON(http.StatusOK, resp)
}

func cleansingObjectWithChangeAntennaType(ctx context.Context, q *query.Queries, antennaID int32, oldType, newType store.AntennaType) error {
	if oldType == newType {
		return nil
	}

	if oldType == store.AntennaTypeUnknown {
		// if oldType is unknown, we don't need anything
		return nil
	}

	switch {
	case oldType == store.AntennaTypePlayer:
		// if oldType is player, we need to delete player, card, and hand
		antenna, err := q.GetAntennaById(ctx, antennaID)
		if err != nil {
			return fmt.Errorf("q.GetAntennaById(): %w", err)
		}
		if !antenna.PlayerID.Valid {
			// if playerID is not set, we don't need to delete player
			return nil
		}

		if err := q.DeletePlayerWithHandWithCards(ctx, antenna.PlayerID.Int32); err != nil {
			return fmt.Errorf("q.DeletePlayerWithHandWithCards(): %w", err)
		}
	case oldType == store.AntennaTypeMuck:
		// if oldType is muck, we need to delete muck
	case oldType == store.AntennaTypeBoard:
		// if oldType is board, we need to delete board
		if err := q.DeleteBoardCards(ctx); err != nil {
			return fmt.Errorf("q.DeleteBoardCards(): %w", err)
		}
	default:
		return errors.New("unknown antenna type")
	}

	return nil
}

func HandleDeleteAdminAntenna(c echo.Context, conn *sql.DB) error {
	logger := slog.With("method", "HandleDeleteAdminAntenna")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.WarnContext(c.Request().Context(), "strconv.Atoi", "error", err, slog.Int("id", id))
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
	}

	tx, err := conn.BeginTx(c.Request().Context(), nil)
	if err != nil {
		logger.WarnContext(c.Request().Context(), "conn.BeginTx", "error", err)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}
	defer func() {
		if r := recover(); r != nil || err != nil {
			tx.Rollback()
		}
	}()
	qWithTx := query.New(tx)

	antenna, err := qWithTx.GetAntennaById(c.Request().Context(), int32(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		}
		slog.WarnContext(c.Request().Context(), "q.GetAntennaById", "error", err, slog.Int("id", id))
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	if err := qWithTx.DeleteAntennaByID(c.Request().Context(), int32(id)); err != nil {
		slog.WarnContext(c.Request().Context(), "q.DeleteAntennaById", "error", err, slog.Int("id", id))
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	if err := cleansingObjectWithChangeAntennaType(
		c.Request().Context(), qWithTx, antenna.ID,
		store.GetAntennaType(antenna.AntennaTypeName),
		store.AntennaTypeUnknown,
	); err != nil {
		slog.WarnContext(c.Request().Context(), "cleansingObjectWithChangeAntennaType", "error", err, slog.Int("id", id))
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	if err := tx.Commit(); err != nil {
		slog.WarnContext(c.Request().Context(), "tx.Commit", "error", err)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	notifyClients()
	return c.JSON(http.StatusNoContent, nil)
}
