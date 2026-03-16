package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"gopkg.in/yaml.v3"
)

const (
	EnvConfigPath     = "RFID_POKER_CONFIG_PATH"
	EnvTestConfigPath = "RFID_POKER_TEST_CONFIG_PATH"
)

// CardConfig holds only the card_ids from config.yaml
type CardConfig struct {
	CardIDs map[string]string `yaml:"card_ids"`
}

// TestConfig defines antennas for testing
type TestConfig struct {
	Antennas []AntennaConfig `yaml:"antennas"`
}

// AntennaConfig defines a single antenna configuration
type AntennaConfig struct {
	Serial string `yaml:"serial"`
	Type   string `yaml:"type"`    // player, board, muck
	PairID int    `yaml:"pair_id"` // pair_id for card requests (default: 1)
}

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	var (
		serverURL      string
		cardConfigPath string
		testConfigPath string
		transport      string
		mqttBroker     string
		mqttPort       int
		mqttUser       string
		mqttPassword   string
	)

	flag.StringVar(&serverURL, "server", "http://localhost:8080", "Server URL (HTTP mode)")
	flag.StringVar(&cardConfigPath, "config", "", "Path to card config.yaml (or set RFID_POKER_CONFIG_PATH)")
	flag.StringVar(&testConfigPath, "test-config", "", "Path to test config.yaml (or set RFID_POKER_TEST_CONFIG_PATH)")
	flag.StringVar(&transport, "transport", "http", "Transport mode: http or mqtt")
	flag.StringVar(&mqttBroker, "mqtt-broker", "localhost", "MQTT broker hostname")
	flag.IntVar(&mqttPort, "mqtt-port", 1883, "MQTT broker port")
	flag.StringVar(&mqttUser, "mqtt-user", "", "MQTT username (optional)")
	flag.StringVar(&mqttPassword, "mqtt-password", "", "MQTT password (optional)")
	flag.Parse()

	// Load card config
	cardConfigFilePath, err := fetchConfigPath(cardConfigPath, EnvConfigPath, "./config.yaml")
	if err != nil {
		return fmt.Errorf("fetchConfigPath(card): %w", err)
	}

	cardConfig, err := loadCardConfig(cardConfigFilePath)
	if err != nil {
		return fmt.Errorf("loadCardConfig(): %w", err)
	}

	// Build reverse lookup map (card name -> UID) and valid cards list
	cardToUID := buildCardToUIDMap(cardConfig.CardIDs)
	validCards := make([]string, 0, len(cardToUID))
	for card := range cardToUID {
		validCards = append(validCards, card)
	}

	// Load test config
	testConfigFilePath, err := fetchConfigPath(testConfigPath, EnvTestConfigPath, "./test-config.yaml")
	if err != nil {
		return fmt.Errorf("fetchConfigPath(test): %w", err)
	}

	testConfig, err := loadTestConfig(testConfigFilePath)
	if err != nil {
		return fmt.Errorf("loadTestConfig(): %w", err)
	}

	// Create sender based on transport mode
	var sender CardSender
	switch transport {
	case "http":
		sender = NewHTTPCardSender(serverURL)
	case "mqtt":
		clientID := fmt.Sprintf("test-wired-client-%d", os.Getpid())
		s, err := NewMQTTCardSender(MQTTConfig{
			Broker:   mqttBroker,
			Port:     mqttPort,
			User:     mqttUser,
			Password: mqttPassword,
			ClientID: clientID,
		})
		if err != nil {
			return fmt.Errorf("failed to create MQTT sender: %w", err)
		}
		sender = s
	default:
		return fmt.Errorf("unknown transport: %s (must be http or mqtt)", transport)
	}
	defer sender.Close()

	// Initialize antenna states (no cards initially - added via TUI)
	antennas := make([]AntennaState, len(testConfig.Antennas))
	for i, cfg := range testConfig.Antennas {
		antennas[i] = AntennaState{
			Config: cfg,
			Cards:  []string{},
		}
	}

	// Send boot for each unique device_id
	if err := sendBootForAntennas(sender, testConfig.Antennas); err != nil {
		log.Printf("Warning: boot send failed: %v", err)
	}

	// Initialize text input
	ti := textinput.New()
	ti.Placeholder = "e.g., As Kh or As,Kh"
	ti.CharLimit = 20
	ti.Width = 25

	model := Model{
		antennas:     antennas,
		cardToUID:    cardToUID,
		validCards:   validCards,
		sender:       sender,
		sendingCards: make(map[string]bool),
		inputMode:    ModeNormal,
		textInput:    ti,
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run program: %w", err)
	}

	return nil
}

// sendBootForAntennas groups antennas by device_id (serial) and sends a boot
// event for each unique device. Each device includes the pair_ids of its antennas.
func sendBootForAntennas(sender CardSender, antennas []AntennaConfig) error {
	// Group pair_ids by serial (device_id)
	devicePairIDs := make(map[string][]int)
	for _, a := range antennas {
		pairID := a.PairID
		if pairID <= 0 {
			pairID = 1
		}
		devicePairIDs[a.Serial] = append(devicePairIDs[a.Serial], pairID)
	}

	ctx := context.Background()
	for deviceID, pairIDs := range devicePairIDs {
		if err := sender.SendBoot(ctx, deviceID, pairIDs); err != nil {
			return fmt.Errorf("SendBoot(%s): %w", deviceID, err)
		}
	}
	return nil
}

func loadCardConfig(configFilePath string) (*CardConfig, error) {
	f, err := os.Open(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	var cfg CardConfig
	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	return &cfg, nil
}

func loadTestConfig(configFilePath string) (*TestConfig, error) {
	f, err := os.Open(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open test config file: %w", err)
	}
	defer f.Close()

	var cfg TestConfig
	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode test config: %w", err)
	}

	return &cfg, nil
}

func fetchConfigPath(override, envVar, defaultPath string) (string, error) {
	input := override
	if input == "" {
		input = os.Getenv(envVar)
	}
	if input == "" {
		return defaultPath, nil
	}

	_, err := os.Stat(input)
	if err == nil {
		return input, nil
	}

	u, err := url.Parse(input)
	if err != nil {
		return "", fmt.Errorf("url.Parse(): %w", err)
	}
	switch u.Scheme {
	case "http", "https":
		return fetchHTTPConfigPath(u)
	default:
		return "", fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}
}

func fetchHTTPConfigPath(u *url.URL) (string, error) {
	dir := os.TempDir()
	p := strings.Split(u.Path, "/")
	fileName := p[len(p)-1]

	fp := filepath.Join(dir, fileName)
	f, err := os.Create(fp)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	resp, err := http.Get(u.String())
	if err != nil {
		return "", fmt.Errorf("failed to get config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get config: status code %d", resp.StatusCode)
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fp, nil
}

func buildCardToUIDMap(cardIDs map[string]string) map[string]string {
	result := make(map[string]string)
	for uid, card := range cardIDs {
		normalizedCard := strings.ToLower(card)
		result[normalizedCard] = uid
	}
	return result
}
