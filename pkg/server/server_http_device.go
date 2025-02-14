package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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

	var registeredAntenna []string

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
			registeredAntenna = append(registeredAntenna, store.ToSerial(input.DeviceID, pairID))
		}
		fmt.Println("registered antenna: ", store.ToSerial(input.DeviceID, pairID))
		fmt.Println("err: ", err)
	}
	if len(registeredAntenna) == 0 {
		return c.JSON(http.StatusOK, "already registered antenna, ok")
	}

	return c.JSON(http.StatusOK, fmt.Sprintf("registered antenna: %v", registeredAntenna))
}
