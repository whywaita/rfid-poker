package playercards

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

type CardConfig struct {
	CardIDs map[string]string `yaml:"card_ids"` // key: uid value: card
}

func ReadConfigPath(args []string) string {
	configPath := "./config.yaml"
	if len(args) != 0 {
		if _, err := os.Stat(args[0]); err != nil {
			return configPath
		}
		configPath = args[0]
	}

	return configPath
}

func LoadConfig(p string) (*CardConfig, error) {
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("os.ReadFile(%s): %w", p, err)
	}

	cc, err := UnmarshalCardConfig(b)
	if err != nil {
		return nil, fmt.Errorf("UnmarshalCardConfig(%s): %w", b, err)
	}

	return cc, nil
}

func UnmarshalCardConfig(in []byte) (*CardConfig, error) {
	var cc CardConfig
	if err := yaml.Unmarshal(in, &cc); err != nil {
		return nil, fmt.Errorf("yaml.Unmarshal(); %w", err)
	}

	return &cc, nil
}

func LoadPlayerCard(in []byte, cc CardConfig) (string, error) {
	str := hex.EncodeToString(append(in[:], 0x00, 0x00, 0x00))

	v, ok := cc.CardIDs[str]
	if !ok {
		return "", fmt.Errorf("unknown card")
	}

	return v, nil
}
