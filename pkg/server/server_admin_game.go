package server

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/whywaita/rfid-poker/pkg/store"

	"github.com/labstack/echo/v4"
)

func HandleDeleteAdminGame(c echo.Context, conn *sql.DB) error {
	logger := slog.With("method", "HandleDeleteAdminGame")
	if err := store.ClearGame(c.Request().Context(), conn); err != nil {
		logger.WarnContext(c.Request().Context(), "failed to delete game", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete game")
	}

	// Reset all antenna type timestamps
	resetAntennaTypeTimestamps()

	notifyClients()

	return c.JSON(http.StatusNoContent, nil)
}
