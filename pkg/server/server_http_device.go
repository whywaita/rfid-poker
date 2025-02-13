package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/whywaita/rfid-poker/pkg/store"
)

type Device struct {
	DeviceID string `json:"device_id"`
	PairIDs  []int  `json:"pair_ids"`
}

// HandleDeviceBoot handle booting device
func HandleDeviceBoot(c echo.Context, conn *sql.DB) error {
	ctx := c.Request().Context()
	defer c.Request().Body.Close()

	input := Device{}
	if err := json.NewDecoder(c.Request().Body).Decode(&input); err != nil {
		log.Printf("invalid request body: %v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	for _, pairID := range input.PairIDs {
		_, err := store.GetAntennaBySerial(ctx, conn, input.DeviceID, pairID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			log.Printf("failed to get antenna: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to get antenna")
		}
		if errors.Is(err, sql.ErrNoRows) {
			err := store.RegisterNewDevice(ctx, conn, input.DeviceID, pairID)
			if err != nil {
				log.Printf("failed to register new antenna: %v", err)
				return echo.NewHTTPError(http.StatusInternalServerError, "failed to register new antenna")
			}
		}
	}

	return c.JSON(http.StatusOK, "success to register new antenna")
}
