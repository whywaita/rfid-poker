package main

import (
	"context"
	"fmt"
	"log"

	"github.com/whywaita/rfid-poker/pkg/server"
)

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	ctx := context.Background()

	if err := server.Run(ctx, "./config.yaml"); err != nil {
		return fmt.Errorf("server.Run(ctx): %w", err)
	}
	return nil
}
