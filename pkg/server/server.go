package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	mysqlmigrate "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/whywaita/rfid-poker/pkg/config"
	"github.com/whywaita/rfid-poker/pkg/query"
	"github.com/whywaita/rfid-poker/pkg/store"
)

type antennaTypeTimestamp struct {
	lastReadTime time.Time
	hasReadCard  bool
}

var (
	antennaTypeTimestamps = map[string]*antennaTypeTimestamp{
		"player": {lastReadTime: time.Time{}, hasReadCard: false},
		"board":  {lastReadTime: time.Time{}, hasReadCard: false},
		"muck":   {lastReadTime: time.Time{}, hasReadCard: false},
	}
	antennaTypeTimestampsMu sync.RWMutex
)

// updateLastCardReadTime updates the timestamp of the last card read for a specific antenna type
func updateLastCardReadTime(antennaType string) {
	antennaTypeTimestampsMu.Lock()
	defer antennaTypeTimestampsMu.Unlock()

	if ts, ok := antennaTypeTimestamps[antennaType]; ok {
		ts.lastReadTime = time.Now()
		ts.hasReadCard = true
	}
}

// getAntennaTypeTimestamps returns a copy of all antenna type timestamps
func getAntennaTypeTimestamps() map[string]antennaTypeTimestamp {
	antennaTypeTimestampsMu.RLock()
	defer antennaTypeTimestampsMu.RUnlock()

	result := make(map[string]antennaTypeTimestamp)
	for k, v := range antennaTypeTimestamps {
		result[k] = *v
	}
	return result
}

// resetAntennaTypeTimestamps resets all antenna type timestamps
func resetAntennaTypeTimestamps() {
	antennaTypeTimestampsMu.Lock()
	defer antennaTypeTimestampsMu.Unlock()

	for _, ts := range antennaTypeTimestamps {
		ts.lastReadTime = time.Time{}
		ts.hasReadCard = false
	}
}

// restoreAntennaTypeTimestamps restores antenna type timestamps from the database on server startup
func restoreAntennaTypeTimestamps(ctx context.Context, conn *sql.DB) error {
	logger := slog.With("method", "restoreAntennaTypeTimestamps")

	q := query.New(conn)

	// Check if there's an active game
	_, err := q.GetCurrentGame(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// No active game, nothing to restore
			logger.InfoContext(ctx, "no active game found, skipping timestamp restoration")
			return nil
		}
		return fmt.Errorf("q.GetCurrentGame(): %w", err)
	}

	// Get all antenna types that have cards in the current game
	antennaTypes, err := q.GetAntennaTypesWithCardsInCurrentGame(ctx)
	if err != nil {
		return fmt.Errorf("q.GetAntennaTypesWithCardsInCurrentGame(): %w", err)
	}

	// Restore timestamps for antenna types that have cards
	antennaTypeTimestampsMu.Lock()
	defer antennaTypeTimestampsMu.Unlock()

	now := time.Now()
	for _, antennaTypeName := range antennaTypes {
		if ts, ok := antennaTypeTimestamps[antennaTypeName]; ok {
			ts.hasReadCard = true
			ts.lastReadTime = now
			logger.InfoContext(ctx, "restored antenna type timestamp",
				"antenna_type", antennaTypeName,
				"last_read_time", now)
		}
	}

	return nil
}

// startGameTimeoutChecker starts a goroutine that checks for game timeout
func startGameTimeoutChecker(ctx context.Context, conn *sql.DB) {
	timeoutSeconds := config.Conf.GameTimeoutSeconds
	if timeoutSeconds <= 0 {
		// Timeout disabled
		slog.InfoContext(ctx, "game timeout is disabled")
		return
	}

	go func() {
		ticker := time.NewTicker(5 * time.Second) // Check every 5 seconds
		defer ticker.Stop()

		slog.InfoContext(ctx, "game timeout checker started", "timeout_seconds", timeoutSeconds)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				timestamps := getAntennaTypeTimestamps()

				// Check if player and board have both read cards
				playerTs, hasPlayer := timestamps["player"]
				boardTs, hasBoard := timestamps["board"]

				// Game must have started: both player and board must have read cards
				if !hasPlayer || !playerTs.hasReadCard || !hasBoard || !boardTs.hasReadCard {
					// Game hasn't properly started yet
					continue
				}

				// Check if all antenna types that have read cards have timed out
				var activeTypes []string      // antenna types that have read cards
				var timedOutTypes []string    // antenna types that have timed out
				var stillActiveTypes []string // antenna types still reading cards

				for antennaType, ts := range timestamps {
					// Skip if no card has been read yet for this antenna type
					if !ts.hasReadCard {
						continue
					}

					activeTypes = append(activeTypes, antennaType)

					elapsed := time.Since(ts.lastReadTime)
					if elapsed >= time.Duration(timeoutSeconds)*time.Second {
						timedOutTypes = append(timedOutTypes, antennaType)
					} else {
						stillActiveTypes = append(stillActiveTypes, antennaType)
					}
				}

				// Only clear game if:
				// 1. Both player and board have read cards (checked above)
				// 2. ALL active antenna types have timed out
				shouldClearGame := len(activeTypes) > 0 && len(stillActiveTypes) == 0

				if shouldClearGame {
					slog.InfoContext(ctx, "game timeout detected, clearing game",
						"timeout_seconds", timeoutSeconds,
						"timed_out_types", timedOutTypes)

					// Clear the game
					if err := store.ClearGame(context.Background(), conn); err != nil {
						slog.WarnContext(ctx, "failed to clear game on timeout", "error", err)
						continue
					}

					// Reset all antenna type timestamps
					resetAntennaTypeTimestamps()

					// Notify clients
					notifyClients()
				}
			}
		}
	}()
}

func Run(ctx context.Context) error {
	go func() {
		runtime.GOMAXPROCS(runtime.NumCPU())
		slog.WarnContext(ctx, http.ListenAndServe("localhost:6060", nil).Error())
	}()

	conn, err := connectMySQL()
	if err != nil {
		return fmt.Errorf("connectMySQL(): %w", err)
	}
	if err := initializeDatabase(conn); err != nil {
		return fmt.Errorf("initializeDatabase(): %w", err)
	}

	// Restore antenna type timestamps from database
	if err := restoreAntennaTypeTimestamps(ctx, conn); err != nil {
		slog.WarnContext(ctx, "failed to restore antenna type timestamps", "error", err)
		// Continue server startup even if restoration fails
	}

	// Start game timeout checker
	startGameTimeoutChecker(ctx, conn)

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.CORSWithConfig(
		middleware.CORSConfig{
			AllowOrigins: []string{"*"},
			AllowMethods: []string{
				http.MethodGet,
				http.MethodHead,
				http.MethodPost,
				http.MethodDelete,
				http.MethodOptions,
			},
			AllowHeaders:     []string{"Content-Type", "Authorization"},
			AllowCredentials: true,
		}))

	// For client
	e.POST("/device/boot", func(c echo.Context) error {
		return HandleDeviceBoot(c, conn)
	})
	e.POST("/card", func(c echo.Context) error {
		return HandleCards(c, conn)
	})

	// For admin
	e.GET("/admin/antenna", func(c echo.Context) error {
		return HandleGetAdminAntenna(c, conn)
	})
	e.POST("/admin/antenna/:id", func(c echo.Context) error {
		return HandlePostAdminAntenna(c, conn)
	})
	e.DELETE("/admin/antenna/:id", func(c echo.Context) error {
		return HandleDeleteAdminAntenna(c, conn)
	})
	e.GET("/admin/player", func(c echo.Context) error {
		return HandleGetAdminPlayers(c, conn)
	})
	e.POST("/admin/player/:id", func(c echo.Context) error {
		return HandlePostAdminPlayer(c, conn)
	})
	e.GET("/admin/player/:id/hand", func(c echo.Context) error {
		return HandleGetAdminPlayerHand(c, conn)
	})
	e.DELETE("/admin/player/:id/hand", func(c echo.Context) error {
		return HandleDeleteAdminPlayerHand(c, conn)
	})
	e.DELETE("/admin/game", func(c echo.Context) error {
		return HandleDeleteAdminGame(c, conn)
	})

	e.GET("/ws", func(c echo.Context) error {
		return ws(c, conn)
	})
	go func() {
		if err := e.Start(":8080"); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.WarnContext(ctx, "failed to start server", "error", err)
			return
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := e.Shutdown(cctx); err != nil {
		e.Logger.Fatal(err)
	}

	return nil
}

func connectMySQL() (*sql.DB, error) {
	cfg := mysql.NewConfig()
	cfg.User = config.Conf.MySQLUser
	cfg.Passwd = config.Conf.MySQLPass
	cfg.Net = "tcp"
	cfg.Addr = fmt.Sprintf("%s:%s", config.Conf.MySQLHost, config.Conf.MySQLPort)
	cfg.DBName = config.Conf.MySQLDatabase

	cfg.MultiStatements = true
	cfg.ParseTime = true

	conn, err := mysql.NewConnector(cfg)
	if err != nil {
		return nil, fmt.Errorf("mysql.NewConnector(): %w", err)
	}

	db := sql.OpenDB(conn)
	return db, nil
}

func initializeDatabase(conn *sql.DB) error {
	driver, err := mysqlmigrate.WithInstance(conn, &mysqlmigrate.Config{})
	if err != nil {
		return fmt.Errorf("mysqlmigrate.WithInstance(): %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://_sqlc/migration",
		"mysql",
		driver,
	)
	if err != nil {
		return fmt.Errorf("migrate.NewWithDatabaseInstance(): %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("m.Up(): %w", err)
	}

	return nil
}
