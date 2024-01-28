package config

import (
	"fmt"
	"log"
	"os"

	"github.com/goccy/go-yaml"
)

type Config struct {
	CardIDs     map[string]string `yaml:"card_ids"` // key: uid value: card
	Players     map[int]string    `yaml:"players"`  // key: serial_number value: player_name
	MuckSerial  string            `yaml:"muck_serial"`
	BoardSerial string            `yaml:"board_serial"`

	// Optional

	HTTPMode bool `yaml:"http_mode"` // if true, use http mode (default: false)
}

func Load(p string) (*Config, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("os.ReadFile(%s): %w", p, err)
	}

	var cc Config
	if err := yaml.Unmarshal(b, &cc); err != nil {
		return nil, fmt.Errorf("yaml.Unmarshal(); %w", err)
	}

	for k, v := range cc.Players {
		log.Printf("loaded player key %v value %s", k, v)
	}

	return &cc, nil
}
