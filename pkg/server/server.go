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
	"strconv"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	_ "github.com/mattn/go-sqlite3"

	"github.com/whywaita/rfid-poker/pkg/config"
	"github.com/whywaita/rfid-poker/pkg/playercards"
	"github.com/whywaita/rfid-poker/pkg/query"
	"github.com/whywaita/rfid-poker/pkg/reader"
	"github.com/whywaita/rfid-poker/pkg/readerhttp"
	"github.com/whywaita/rfid-poker/pkg/readerpasori"
)

func Run(ctx context.Context, configPath string) error {
	go func() {
		runtime.GOMAXPROCS(runtime.NumCPU())
		log.Println(http.ListenAndServe(":6060", nil))
	}()

	handCh := make(chan playercards.HandData)
	deviceCh := make(chan reader.Data)
	updatedCh := make(chan struct{})

	c, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("playercards.LoadConfig(%s): %w", configPath, err)
	}

	conn, err := connectSQLite()
	if err != nil {
		return fmt.Errorf("connectSQLite(): %w", err)
	}
	if err := initializeDatabase(ctx, conn, *c); err != nil {
		return fmt.Errorf("initializeDatabase(): %w", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("conn.Close(): %v", err)
		}
		if err := os.Remove("./instance.db"); err != nil {
			log.Printf("os.Remove(): %v", err)
		}
	}()

	if c.HTTPMode {
		go func() {
			if err := readerhttp.PollingHTTP(deviceCh); err != nil {
				log.Printf("reader.PollingHTTP(): %v", err)
				return
			}
		}()
	} else {
		go func() {
			if err := readerpasori.PollingDevices(deviceCh); err != nil {
				log.Printf("reader.PollingDevices(): %v", err)
				return
			}
		}()
	}
	go func() {
		log.Printf("Start loading cards...")
		if err := playercards.LoadCardsWithChannel(*c, handCh, deviceCh); err != nil {
			log.Printf("playercards.LoadCardsWithChannel(ctx): %v", err)
			return
		}
	}()
	go func() {
		if err := ReceiveData(ctx, conn, handCh, updatedCh, *c); err != nil {
			log.Printf("ReceiveData(): %v", err)
			return
		}
	}()

	e := echo.New()
	e.Use(middleware.Logger())
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}

	return nil
}

func connectSQLite() (*sql.DB, error) {
	instanceFilePath := "./instance.db"

	if _, err := os.Stat(instanceFilePath); os.IsNotExist(err) {
		log.Printf("instance.db is not exist. create new instance.db")
		if _, err := os.Create(instanceFilePath); err != nil {
			return nil, fmt.Errorf("os.Create(%s): %w", instanceFilePath, err)
		}
	}

	conn, err := sql.Open("sqlite3", instanceFilePath)
	if err != nil {
		return nil, fmt.Errorf("sql.Open(): %w", err)
	}

	return conn, nil
}

func initializeDatabase(ctx context.Context, conn *sql.DB, cc config.Config) error {
	db := query.New(conn)

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

	for serial, name := range cc.Players {
		if _, err := db.AddPlayer(ctx, query.AddPlayerParams{
			Name:   name,
			Serial: strconv.Itoa(serial),
		}); err != nil {
			return fmt.Errorf("db.AddPlayer(): %w", err)
		}
	}

	return nil
}
