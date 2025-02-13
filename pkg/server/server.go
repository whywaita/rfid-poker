package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	_ "github.com/mattn/go-sqlite3"
)

const dbPath = "./instance.db"

func Run(ctx context.Context) error {
	go func() {
		runtime.GOMAXPROCS(runtime.NumCPU())
		log.Println(http.ListenAndServe(":6060", nil))
	}()

	updatedCh := make(chan struct{})

	conn, err := connectSQLite()
	if err != nil {
		return fmt.Errorf("connectSQLite(): %w", err)
	}
	if err := initializeDatabase(conn); err != nil {
		return fmt.Errorf("initializeDatabase(): %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("conn.Close(): %v", err)
		}
		if err := os.Remove(dbPath); err != nil {
			log.Printf("os.Remove(): %v", err)
		}
	}()

	e := echo.New()
	e.Use(middleware.Logger())

	e.POST("/device/boot", func(c echo.Context) error {
		return HandleDeviceBoot(c, conn)
	})
	e.POST("/card", func(c echo.Context) error {
		return HandleCards(c, conn, updatedCh)
	})

	e.GET("/ws", func(c echo.Context) error {
		return ws(c, conn, updatedCh)
	})
	go func() {
		if err := e.Start(":8080"); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("failed to start server: %v", err)
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

func connectSQLite() (*sql.DB, error) {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		log.Printf("instance.db is not exist. create new instance.db")
		if _, err := os.Create(dbPath); err != nil {
			return nil, fmt.Errorf("os.Create(%s): %w", dbPath, err)
		}
	}

	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("sql.Open(): %w", err)
	}

	return conn, nil
}

func initializeDatabase(conn *sql.DB) error {
	driver, err := sqlite3.WithInstance(conn, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("sqlite3.WithInstance(): %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://_sqlc/migration",
		"sqlite3",
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
