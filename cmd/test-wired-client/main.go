package main

import (
	"bufio"
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"gopkg.in/yaml.v3"

	"github.com/whywaita/rfid-poker/pkg/version"
)

// PostCardRequest represents the request body for POST /card endpoint
type PostCardRequest struct {
	UID      string `json:"uid"`
	DeviceID string `json:"device_id"`
	PairID   int    `json:"pair_id"`
}

// ConfigFile represents the minimal config.yaml structure we need
type ConfigFile struct {
	CardIDs map[string]string `yaml:"card_ids"`
}

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	var configPath string
	var serial string
	var antennaType string
	var serverURL string
	var pairID int
	var debug bool

	flag.StringVar(&configPath, "config", "./config.yaml", "Path to config.yaml file")
	flag.StringVar(&serial, "serial", "", "Antenna serial/device ID (required)")
	flag.StringVar(&antennaType, "type", "player", "Antenna type (player, board, muck)")
	flag.StringVar(&serverURL, "server", "http://localhost:8080", "Server URL for HTTP POST")
	flag.IntVar(&pairID, "pair-id", 1, "Antenna pair ID")
	flag.BoolVar(&debug, "debug", false, "Enable debug logging")
	flag.Parse()

	logLevel := slog.LevelInfo
	if debug {
		logLevel = slog.LevelDebug
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: false,
		Level:     logLevel,
	})))

	slog.Info("Starting test-wired-client",
		slog.String("version", version.GetVersion()),
		slog.String("commit", version.GetCommit()),
	)

	if serial == "" {
		return fmt.Errorf("serial is required. Use -serial flag")
	}

	// Load config.yaml
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg ConfigFile
	if err := yaml.Unmarshal(configData, &cfg); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	if len(cfg.CardIDs) == 0 {
		return fmt.Errorf("no cards found in config file")
	}

	slog.Info("Configuration loaded",
		slog.String("config_path", configPath),
		slog.Int("card_count", len(cfg.CardIDs)),
	)

	// Create a reverse map: card name -> UID
	cardNameToUID := make(map[string]string)
	for uid, cardName := range cfg.CardIDs {
		cardNameToUID[strings.ToLower(cardName)] = uid
	}

	fmt.Printf("Test Wired Client\n")
	fmt.Printf("=================\n")
	fmt.Printf("Serial:       %s\n", serial)
	fmt.Printf("Type:         %s\n", antennaType)
	fmt.Printf("Pair ID:      %d\n", pairID)
	fmt.Printf("Server:       %s\n", serverURL)
	fmt.Printf("Config:       %s\n", configPath)
	fmt.Printf("Cards loaded: %d\n", len(cfg.CardIDs))
	fmt.Printf("\n")
	fmt.Printf("Commands:\n")
	fmt.Printf("  - Enter card name (e.g., 'As', 'Kh', '2d') to send card\n")
	fmt.Printf("  - Enter card UID directly to send card\n")
	fmt.Printf("  - Enter 'list' to show all available cards\n")
	fmt.Printf("  - Enter 'quit' or 'exit' to quit\n")
	fmt.Printf("\n")

	scanner := bufio.NewScanner(os.Stdin)
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Handle special commands
		switch strings.ToLower(input) {
		case "quit", "exit":
			fmt.Println("Exiting...")
			return nil
		case "list":
			listCards(cfg.CardIDs)
			continue
		}

		// Try to find the card UID
		var uid string
		inputLower := strings.ToLower(input)

		// Check if input is a card name (e.g., As, Kh)
		if mappedUID, ok := cardNameToUID[inputLower]; ok {
			uid = mappedUID
			slog.Debug("Resolved card name to UID",
				slog.String("card_name", input),
				slog.String("uid", uid),
			)
		} else if _, ok := cfg.CardIDs[input]; ok {
			// Input is already a UID
			uid = input
		} else {
			fmt.Printf("❌ Unknown card: %s\n", input)
			fmt.Println("   Use 'list' to see all available cards")
			continue
		}

		// Send card to server
		if err := sendCard(httpClient, serverURL, serial, pairID, uid); err != nil {
			fmt.Printf("❌ Failed to send card: %v\n", err)
			slog.Error("Failed to send card",
				slog.String("uid", uid),
				slog.String("error", err.Error()),
			)
		} else {
			cardName := cfg.CardIDs[uid]
			fmt.Printf("✅ Card sent: %s (UID: %s)\n", cardName, uid)
			slog.Info("Card sent successfully",
				slog.String("card_name", cardName),
				slog.String("uid", uid),
			)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	return nil
}

func sendCard(client *http.Client, serverURL, deviceID string, pairID int, uid string) error {
	req := PostCardRequest{
		UID:      uid,
		DeviceID: deviceID,
		PairID:   pairID,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", serverURL+"/card", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNotModified {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func listCards(cardIDs map[string]string) {
	fmt.Println("\nAvailable cards:")
	fmt.Println("================")

	// Organize cards by suit
	suits := []string{"s", "h", "d", "c"}
	suitNames := map[string]string{
		"s": "♠ Spades",
		"h": "♥ Hearts",
		"d": "♦ Diamonds",
		"c": "♣ Clubs",
	}

	for _, suit := range suits {
		fmt.Printf("\n%s:\n", suitNames[suit])
		for uid, cardName := range cardIDs {
			if strings.HasSuffix(strings.ToLower(cardName), suit) {
				fmt.Printf("  %-4s -> %s\n", cardName, uid)
			}
		}
	}
	fmt.Println()
}
