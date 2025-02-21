package server

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/whywaita/rfid-poker/pkg/store"

	"github.com/labstack/echo/v4"
)

func HandleDeleteAdminGame(c echo.Context, conn *sql.DB, updatedCh chan struct{}) error {
	if err := store.ClearGame(c.Request().Context(), conn); err != nil {
		log.Printf("failed to delete game: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to delete game")
	}

	updatedCh <- struct{}{}

	return c.JSON(http.StatusNoContent, nil)
}
