package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jinzhu/configor"

	"github.com/whywaita/rfid-poker/pkg/config"
	"github.com/whywaita/rfid-poker/pkg/server"
)

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	ctx := context.Background()

	if err := configor.Load(&config.Conf, "config.yaml"); err != nil {
		return fmt.Errorf("configor.Load(): %w", err)
	}

	if err := server.Run(ctx); err != nil {
		return fmt.Errorf("server.Run(ctx): %w", err)
	}
	return nil
}
