package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
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
	logger := slog.With("method", "HandleDeviceBoot")
	ctx := c.Request().Context()
	defer c.Request().Body.Close()

	input := Device{}
	if err := json.NewDecoder(c.Request().Body).Decode(&input); err != nil {
		logger.WarnContext(ctx, "invalid request body", "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	logger = logger.With("device_id", input.DeviceID)

	var registeredAntenna []string
	for _, pairID := range input.PairIDs {
		logger = logger.With("pair_id", pairID)
		_, err := store.GetAntennaBySerial(ctx, conn, input.DeviceID, pairID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			logger.WarnContext(ctx, "failed to get antenna", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to get antenna")
		}
		if errors.Is(err, sql.ErrNoRows) {
			err := store.RegisterNewDevice(ctx, conn, input.DeviceID, pairID)
			if err != nil {
				logger.WarnContext(ctx, "failed to register new antenna", "error", err)
				return echo.NewHTTPError(http.StatusInternalServerError, "failed to register new antenna")
			}
			registeredAntenna = append(registeredAntenna, store.ToSerial(input.DeviceID, pairID))
		}
	}
	if len(registeredAntenna) == 0 {
		return c.JSON(http.StatusOK, "already registered antenna, ok")
	}

	return c.JSON(http.StatusOK, fmt.Sprintf("registered antenna: %v", registeredAntenna))
}
