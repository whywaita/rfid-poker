package config

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	CardIDs map[string]string `yaml:"card_ids"` // key: uid value: card
	Players map[string]string `yaml:"players"`  // key: serial_number value: player_name
	//MuckSerial string            `yaml:"muck_serial"`
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
