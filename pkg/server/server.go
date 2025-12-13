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

// antennaTimestamp tracks the last card read time for a specific antenna (by serial number)
type antennaTimestamp struct {
	lastReadTime time.Time
	hasReadCard  bool
	antennaType  string // "player", "board", or "muck"
}

var (
	// antennaTimestamps maps antenna serial number to its timestamp info
	antennaTimestamps   = make(map[string]*antennaTimestamp)
	antennaTimestampsMu sync.RWMutex
)

// updateLastCardReadTime updates the timestamp of the last card read for a specific antenna
func updateLastCardReadTime(serial, antennaType string) {
	antennaTimestampsMu.Lock()
	defer antennaTimestampsMu.Unlock()

	if ts, ok := antennaTimestamps[serial]; ok {
		ts.lastReadTime = time.Now()
		ts.hasReadCard = true
	} else {
		antennaTimestamps[serial] = &antennaTimestamp{
			lastReadTime: time.Now(),
			hasReadCard:  true,
			antennaType:  antennaType,
		}
	}
}

// getAntennaTimestamps returns a copy of all antenna timestamps
func getAntennaTimestamps() map[string]antennaTimestamp {
	antennaTimestampsMu.RLock()
	defer antennaTimestampsMu.RUnlock()

	result := make(map[string]antennaTimestamp)
	for k, v := range antennaTimestamps {
		result[k] = *v
	}
	return result
}

// resetAntennaTimestamps resets all antenna timestamps
func resetAntennaTimestamps() {
	antennaTimestampsMu.Lock()
	defer antennaTimestampsMu.Unlock()

	antennaTimestamps = make(map[string]*antennaTimestamp)
}

// removeAntennaTimestamp removes a specific antenna from timestamp tracking
func removeAntennaTimestamp(serial string) {
	antennaTimestampsMu.Lock()
	defer antennaTimestampsMu.Unlock()

	delete(antennaTimestamps, serial)
}

// restoreAntennaTimestamps restores antenna timestamps from the database on server startup
func restoreAntennaTimestamps(ctx context.Context, conn *sql.DB) error {
	logger := slog.With("method", "restoreAntennaTimestamps")

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

	// Get all antennas that have cards in the current game
	antennas, err := q.GetAntennasWithCardsInCurrentGame(ctx)
	if err != nil {
		return fmt.Errorf("q.GetAntennasWithCardsInCurrentGame(): %w", err)
	}

	// Restore timestamps for antennas that have cards
	antennaTimestampsMu.Lock()
	defer antennaTimestampsMu.Unlock()

	now := time.Now()
	for _, antenna := range antennas {
		antennaTimestamps[antenna.Serial] = &antennaTimestamp{
			lastReadTime: now,
			hasReadCard:  true,
			antennaType:  antenna.AntennaTypeName,
		}
		logger.InfoContext(ctx, "restored antenna timestamp",
			"serial", antenna.Serial,
			"antenna_type", antenna.AntennaTypeName,
			"last_read_time", now)
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
				timestamps := getAntennaTimestamps()

				// Categorize antennas by type
				var activePlayerAntennas []string   // player antennas that have read cards
				var timedOutPlayerAntennas []string // player antennas that have timed out
				var hasActiveBoard bool             // whether board has active cards

				for serial, ts := range timestamps {
					if !ts.hasReadCard {
						continue
					}

					elapsed := time.Since(ts.lastReadTime)
					isTimedOut := elapsed >= time.Duration(timeoutSeconds)*time.Second

					switch ts.antennaType {
					case "player":
						activePlayerAntennas = append(activePlayerAntennas, serial)
						if isTimedOut {
							timedOutPlayerAntennas = append(timedOutPlayerAntennas, serial)
						}
					case "board":
						if !isTimedOut {
							hasActiveBoard = true
						}
					}
				}

				// Game must have started: at least one player and board must have read cards
				if len(activePlayerAntennas) == 0 || !hasActiveBoard {
					// Game hasn't properly started yet
					continue
				}

				// Remove timed out players from timestamp tracking (but don't muck them)
				// Muck should only happen via muck antenna, not via timeout
				for _, serial := range timedOutPlayerAntennas {
					slog.InfoContext(ctx, "player antenna timed out, removing from tracking",
						"serial", serial,
						"timeout_seconds", timeoutSeconds)

					// Remove from timestamp tracking (don't muck - that's only for muck antenna)
					removeAntennaTimestamp(serial)
				}

				// Re-check active players after removal
				timestamps = getAntennaTimestamps()
				activePlayerCount := 0
				for _, ts := range timestamps {
					if ts.hasReadCard && ts.antennaType == "player" {
						activePlayerCount++
					}
				}

				// Clear game if all players have timed out
				if activePlayerCount == 0 && len(activePlayerAntennas) > 0 {
					slog.InfoContext(ctx, "all players timed out, clearing game",
						"timeout_seconds", timeoutSeconds,
						"timed_out_players", timedOutPlayerAntennas)

					// Clear the game
					if err := store.ClearGame(context.Background(), conn); err != nil {
						slog.WarnContext(ctx, "failed to clear game on timeout", "error", err)
						continue
					}

					// Reset all antenna timestamps
					resetAntennaTimestamps()

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
	if err := restoreAntennaTimestamps(ctx, conn); err != nil {
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
