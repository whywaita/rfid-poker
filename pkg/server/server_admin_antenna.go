package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/whywaita/rfid-poker/pkg/store"

	"github.com/labstack/echo/v4"
	"github.com/whywaita/rfid-poker/pkg/query"
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
	q := query.New(conn)

	antenna, err := q.GetAntenna(c.Request().Context())
	if err != nil {
		log.Printf("q.GetAdminAntenna(): %v", err)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	var respAntenna []Antenna
	for _, a := range antenna {
		deviceID, pairID, err := store.FromSerial(a.Serial)
		if err != nil {
			log.Printf("store.FromSerial(%s): %v", a.Serial, err)
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
	q := query.New(conn)

	var req PostAdminAntennaRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("c.Bind(): %v", err)
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
	}

	id, err := strconv.Atoi(req.ID)
	if err != nil {
		log.Printf("strconv.Atoi(%s): %v", req.ID, err)
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
	}

	antenna, err := q.GetAntennaById(c.Request().Context(), int32(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		}
		log.Printf("store.GetUnknownAntennaTypeID(): %v", err)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	// check antenna type name
	if store.GetAntennaType(req.AntennaTypeName) == store.AntennaTypeUnknown {
		log.Printf("antenna type name %s is unknown", req.AntennaTypeName)
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: fmt.Sprintf("antenna type name (input: %s) is unknown", req.AntennaTypeName)})
	}

	if _, err := q.GetAntennaTypeIdByAntennaTypeName(c.Request().Context(), req.AntennaTypeName); err != nil {
		log.Printf("q.GetAntennaTypeIdByAntennaTypeName(): %v", err)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	if _, err := q.SetAntennaTypeToAntennaBySerial(c.Request().Context(), query.SetAntennaTypeToAntennaBySerialParams{
		Name:   req.AntennaTypeName,
		Serial: antenna.Serial,
	}); err != nil {
		log.Printf("q.SetAntennaTypeToAntennaBySerial(): %v", err)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	if err := cleansingObjectWithChangeAntennaType(
		c.Request().Context(), conn, antenna.ID,
		store.GetAntennaType(antenna.AntennaTypeName),
		store.GetAntennaType(req.AntennaTypeName),
	); err != nil {
		log.Printf("cleansingObjectWithChangeAntennaType(): %v", err)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	respAntenna, err := q.GetAntennaById(c.Request().Context(), int32(id))
	if err != nil {
		log.Printf("q.GetAntennaById(): %v", err)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	deviceID, pairID, err := store.FromSerial(respAntenna.Serial)
	if err != nil {
		log.Printf("store.FromSerial(%s): %v", respAntenna.Serial, err)
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

func cleansingObjectWithChangeAntennaType(ctx context.Context, conn *sql.DB, antennaID int32, oldType, newType store.AntennaType) error {
	if oldType == newType {
		return nil
	}

	if oldType == store.AntennaTypeUnknown {
		// if oldType is unknown, we don't need anything
		return nil
	}

	q := query.New(conn)

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
