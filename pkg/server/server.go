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
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	mysqlmigrate "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/whywaita/rfid-poker/pkg/config"
)

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
