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

type GetAdminAntennaResponse struct {
	Antenna []GetAdminAntennaResponseAntenna `json:"antenna"`
}

type GetAdminAntennaResponseAntenna struct {
	ID              int64  `json:"id"`
	DeviceID        string `json:"device_id"`
	PairID          int    `json:"pair_id"`
	AntennaTypeName string `json:"antenna_type_name"`
}

func HandleGetAdminAntenna(c echo.Context, conn *sql.DB) error {
	q := query.New(conn)

	antenna, err := q.GetAntenna(c.Request().Context())
	if err != nil {
		log.Printf("q.GetAdminAntenna(): %v", err)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
	}

	var respAntenna []GetAdminAntennaResponseAntenna
	for _, a := range antenna {
		deviceID, pairID, err := store.FromSerial(a.Serial)
		if err != nil {
			log.Printf("store.FromSerial(%s): %v", a.Serial, err)
			return c.JSON(http.StatusInternalServerError, nil)
		}
		respAntenna = append(respAntenna, GetAdminAntennaResponseAntenna{
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
	ID              string `param:"id"`
	AntennaTypeName string `json:"antenna_type_name"`
}

func HandlePostAdminAntenna(c echo.Context, conn *sql.DB) error {
	q := query.New(conn)

	//inputID := c.Param("id")
	//// convert to int64
	//id, err := strconv.Atoi(inputID)
	//if err != nil {
	//	log.Printf("strconv.Atoi(%s): %v", inputID, err)
	//	return c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
	//}

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

	antenna, err := q.GetAntennaById(c.Request().Context(), int64(id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.JSON(http.StatusNotFound, nil)
		}
		log.Printf("store.GetUnknownAntennaTypeID(): %v", err)
		return c.JSON(http.StatusInternalServerError, nil)
	}

	if _, err := q.GetAntennaTypeIdByAntennaTypeName(c.Request().Context(), req.AntennaTypeName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Printf("antenna type name %s is not found", req.AntennaTypeName)
			return c.JSON(http.StatusBadRequest, nil)
		}
		log.Printf("q.GetAntennaTypeIdByAntennaTypeName(): %v", err)
		return c.JSON(http.StatusInternalServerError, nil)
	}

	if _, err := q.SetAntennaTypeToAntennaBySerial(c.Request().Context(), query.SetAntennaTypeToAntennaBySerialParams{
		Name:   req.AntennaTypeName,
		Serial: antenna.Serial,
	}); err != nil {
		log.Printf("q.SetAntennaTypeToAntennaBySerial(): %v", err)
		return c.JSON(http.StatusInternalServerError, nil)
	}

	return c.JSON(http.StatusOK, nil)
}
