package server

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/whywaita/rfid-poker/pkg/query"

	"github.com/whywaita/rfid-poker/pkg/config"
	"github.com/whywaita/rfid-poker/pkg/playercards"
)

func ReceiveData(ctx context.Context, conn *sql.DB, ch chan playercards.HandData, updateCh chan struct{}, cc config.Config) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case v := <-ch:
			go func() {
				if err := process(ctx, conn, v, updateCh, cc); err != nil {
					log.Printf("Error process(): %v", err)
				}
			}()
		}
	}
}

func process(ctx context.Context, conn *sql.DB, v playercards.HandData, updateCh chan struct{}, cc config.Config) error {
	log.Printf("receive card in server: %s", v)
	if len(v.Cards) < 1 && len(v.Cards) > 3 {
		return fmt.Errorf("invalid card: %s", v)
	}

	q := query.New(conn)

	switch {
	case strings.EqualFold(v.SerialNumber, cc.MuckSerial):
		log.Printf("MuckPlayer(): %s", v)
		if err := MuckPlayer(ctx, q, v.Cards); err != nil {
			return fmt.Errorf("MuckPlayer(): %w", err)
		}
	case strings.EqualFold(v.SerialNumber, cc.BoardSerial):
		log.Printf("AddBoard(): %s", v)
		if err := AddBoard(ctx, q, v.Cards); err != nil {
			return fmt.Errorf("AddBoard(): %w", err)
		}
	default:
		log.Printf("AddPlayer(): %s", v)
		err := AddPlayer(ctx, conn, v.Cards, v.SerialNumber)
		if err != nil {
			return fmt.Errorf("AddPlayer(): %w", err)
		}
	}

	updateCh <- struct{}{}
	return nil
}
