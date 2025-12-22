package main

import (
	"bytes"
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
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/goccy/go-json"
	"gopkg.in/yaml.v3"

	"github.com/whywaita/rfid-poker/pkg/version"
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

// PostCardRequest represents the request body for POST /card endpoint
type PostCardRequest struct {
	UID      string `json:"uid"`
	DeviceID string `json:"device_id"`
	PairID   int    `json:"pair_id"`
}

// AntennaState tracks runtime state for an antenna
type AntennaState struct {
	Config     AntennaConfig
	Cards      []string // cards assigned to this antenna (editable in TUI)
	IsActive   bool     // whether cards are being "read"
	LastStatus string   // last HTTP response status
	LastError  string   // last error message
}

// InputMode represents the current input mode
type InputMode int

const (
	ModeNormal InputMode = iota
	ModeAddCard
)

// Model is the bubbletea model
type Model struct {
	antennas     []AntennaState
	cardToUID    map[string]string
	validCards   []string // list of valid card names for autocomplete hint
	serverURL    string
	client       *http.Client
	selectedIdx  int
	quitting     bool
	width        int
	height       int
	lastMessage  string
	sendingCards map[string]bool // track which antennas are currently sending
	inputMode    InputMode
	textInput    textinput.Model
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57"))

	activeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	inactiveStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)

	inputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)

	cardStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("213"))
)

// Key bindings for normal mode
type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Toggle   key.Binding
	AddCard  key.Binding
	DelCard  key.Binding
	ClearAll key.Binding
	Quit     key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Toggle: key.NewBinding(
		key.WithKeys("enter", " "),
		key.WithHelp("enter/space", "toggle"),
	),
	AddCard: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "add card"),
	),
	DelCard: key.NewBinding(
		key.WithKeys("d", "x"),
		key.WithHelp("d/x", "delete last card"),
	),
	ClearAll: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "clear all cards"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

// Messages
type cardSentMsg struct {
	antennaIdx int
	success    bool
	status     string
	err        error
}

type tickMsg time.Time

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	var serverURL string
	var cardConfigPath string
	var testConfigPath string

	flag.StringVar(&serverURL, "server", "http://localhost:8080", "Server URL")
	flag.StringVar(&cardConfigPath, "config", "", "Path to card config.yaml (or set RFID_POKER_CONFIG_PATH)")
	flag.StringVar(&testConfigPath, "test-config", "", "Path to test config.yaml (or set RFID_POKER_TEST_CONFIG_PATH)")
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

	// Initialize antenna states (no cards initially - added via TUI)
	antennas := make([]AntennaState, len(testConfig.Antennas))
	for i, cfg := range testConfig.Antennas {
		antennas[i] = AntennaState{
			Config:   cfg,
			Cards:    []string{},
			IsActive: false,
		}
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
		serverURL:    serverURL,
		client:       &http.Client{Timeout: 5 * time.Second},
		selectedIdx:  0,
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

func (m Model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), tea.EnterAltScreen)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle input mode
		if m.inputMode == ModeAddCard {
			switch msg.String() {
			case "esc":
				m.inputMode = ModeNormal
				m.textInput.Reset()
				m.lastMessage = "Cancelled"
				return m, nil
			case "enter":
				input := strings.TrimSpace(m.textInput.Value())
				if input != "" {
					antenna := &m.antennas[m.selectedIdx]
					maxCards := getMaxCards(antenna.Config.Type)

					// Check if already at max
					if maxCards > 0 && len(antenna.Cards) >= maxCards {
						m.lastMessage = errorStyle.Render(fmt.Sprintf("Max %d cards for %s antenna", maxCards, antenna.Config.Type))
						m.inputMode = ModeNormal
						m.textInput.Reset()
						return m, nil
					}

					// Split by comma or space to support multiple cards
					// e.g., "As,Kh" or "As Kh" or "As, Kh"
					input = strings.ReplaceAll(input, ",", " ")
					cardInputs := strings.Fields(input)

					var addedCards []string
					var invalidCards []string
					var duplicateCards []string
					var skippedMax bool

					for _, card := range cardInputs {
						card = strings.TrimSpace(card)
						if card == "" {
							continue
						}

						// Check max cards limit
						if maxCards > 0 && len(antenna.Cards) >= maxCards {
							skippedMax = true
							break
						}

						// Lookup uses lowercase
						lookupKey := strings.ToLower(card)
						if _, ok := m.cardToUID[lookupKey]; !ok {
							invalidCards = append(invalidCards, card)
							continue
						}

						// Check for duplicate cards across all antennas (skip for muck)
						displayCard := normalizeCardName(card)
						if antenna.Config.Type != "muck" && m.isCardInUse(displayCard) {
							duplicateCards = append(duplicateCards, displayCard)
							continue
						}

						// Store with normalized format (As, Kh, etc.)
						antenna.Cards = append(antenna.Cards, displayCard)
						addedCards = append(addedCards, displayCard)
					}

					// Build result message
					var msgParts []string
					if len(addedCards) > 0 {
						msgParts = append(msgParts, fmt.Sprintf("Added: %s", strings.Join(addedCards, ", ")))
					}
					if len(invalidCards) > 0 {
						msgParts = append(msgParts, errorStyle.Render(fmt.Sprintf("Invalid: %s", strings.Join(invalidCards, ", "))))
					}
					if len(duplicateCards) > 0 {
						msgParts = append(msgParts, errorStyle.Render(fmt.Sprintf("Duplicate: %s", strings.Join(duplicateCards, ", "))))
					}
					if skippedMax {
						msgParts = append(msgParts, errorStyle.Render(fmt.Sprintf("(max %d cards)", maxCards)))
					}
					if len(msgParts) > 0 {
						m.lastMessage = strings.Join(msgParts, " | ")
					}
				}
				m.inputMode = ModeNormal
				m.textInput.Reset()
				return m, nil
			default:
				var cmd tea.Cmd
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}
		}

		// Normal mode key handling
		switch {
		case key.Matches(msg, keys.Quit):
			m.quitting = true
			return m, tea.Quit
		case key.Matches(msg, keys.Up):
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}
		case key.Matches(msg, keys.Down):
			if m.selectedIdx < len(m.antennas)-1 {
				m.selectedIdx++
			}
		case key.Matches(msg, keys.Toggle):
			if m.selectedIdx < len(m.antennas) {
				m.antennas[m.selectedIdx].IsActive = !m.antennas[m.selectedIdx].IsActive
				antenna := &m.antennas[m.selectedIdx]
				if antenna.IsActive {
					if len(antenna.Cards) == 0 {
						m.lastMessage = errorStyle.Render("No cards to send! Add cards first with 'a'")
						antenna.IsActive = false
					} else {
						m.lastMessage = fmt.Sprintf("Activating %s...", antenna.Config.Serial)
						m.sendingCards[antenna.Config.Serial] = true
						return m, m.sendCardsCmd(m.selectedIdx, antenna.Config.Serial, antenna.Cards, antenna.Config.PairID)
					}
				} else {
					m.lastMessage = fmt.Sprintf("Deactivated %s", antenna.Config.Serial)
				}
			}
		case key.Matches(msg, keys.AddCard):
			m.inputMode = ModeAddCard
			m.textInput.Focus()
			m.lastMessage = "Enter cards (e.g., As Kh or As,Kh)"
			return m, textinput.Blink
		case key.Matches(msg, keys.DelCard):
			if m.selectedIdx < len(m.antennas) {
				antenna := &m.antennas[m.selectedIdx]
				if len(antenna.Cards) > 0 {
					removed := antenna.Cards[len(antenna.Cards)-1]
					antenna.Cards = antenna.Cards[:len(antenna.Cards)-1]
					m.lastMessage = fmt.Sprintf("Removed card: %s", removed)
				} else {
					m.lastMessage = "No cards to remove"
				}
			}
		case key.Matches(msg, keys.ClearAll):
			if m.selectedIdx < len(m.antennas) {
				antenna := &m.antennas[m.selectedIdx]
				count := len(antenna.Cards)
				antenna.Cards = []string{}
				antenna.IsActive = false
				m.lastMessage = fmt.Sprintf("Cleared %d cards", count)
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		// Resend cards for active antennas
		var cmds []tea.Cmd
		for i, antenna := range m.antennas {
			if antenna.IsActive && !m.sendingCards[antenna.Config.Serial] && len(antenna.Cards) > 0 {
				// Mark as sending before launching goroutine to prevent duplicate sends
				m.sendingCards[antenna.Config.Serial] = true
				cmds = append(cmds, m.sendCardsCmd(i, antenna.Config.Serial, antenna.Cards, antenna.Config.PairID))
			}
		}
		cmds = append(cmds, tickCmd())
		return m, tea.Batch(cmds...)

	case cardSentMsg:
		if msg.antennaIdx < len(m.antennas) {
			antenna := &m.antennas[msg.antennaIdx]
			delete(m.sendingCards, antenna.Config.Serial)
			if msg.err != nil {
				antenna.LastError = msg.err.Error()
				antenna.LastStatus = ""
			} else {
				antenna.LastStatus = msg.status
				antenna.LastError = ""
			}
		}
	}

	return m, nil
}

// isCardInUse checks if a card is already in use by any antenna
func (m *Model) isCardInUse(card string) bool {
	normalizedCard := strings.ToLower(card)
	for _, antenna := range m.antennas {
		for _, c := range antenna.Cards {
			if strings.ToLower(c) == normalizedCard {
				return true
			}
		}
	}
	return false
}

func (m *Model) sendCardsCmd(antennaIdx int, serial string, cards []string, pairID int) tea.Cmd {
	// Copy data needed for the goroutine to avoid race conditions
	cardToUID := m.cardToUID
	client := m.client
	serverURL := m.serverURL

	// Make a copy of cards slice
	cardsCopy := make([]string, len(cards))
	copy(cardsCopy, cards)

	// Default pair_id to 1 if not set
	if pairID <= 0 {
		pairID = 1
	}

	return func() tea.Msg {
		for _, card := range cardsCopy {
			uid, ok := cardToUID[strings.ToLower(card)]
			if !ok {
				return cardSentMsg{
					antennaIdx: antennaIdx,
					success:    false,
					err:        fmt.Errorf("card %s not found", card),
				}
			}

			err := sendCard(client, serverURL, uid, serial, pairID)
			if err != nil {
				return cardSentMsg{
					antennaIdx: antennaIdx,
					success:    false,
					err:        err,
				}
			}
		}

		return cardSentMsg{
			antennaIdx: antennaIdx,
			success:    true,
			status:     "OK",
		}
	}
}

func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("🃏 Test Wired Client"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("   Server: %s\n", m.serverURL))
	b.WriteString(fmt.Sprintf("   Version: %s (%s)\n\n", version.GetVersion(), version.GetCommit()))

	// Antennas
	for i, antenna := range m.antennas {
		isSelected := i == m.selectedIdx

		// Status indicator
		var statusIndicator string
		if antenna.IsActive {
			statusIndicator = activeStyle.Render("● ACTIVE")
		} else {
			statusIndicator = inactiveStyle.Render("○ inactive")
		}

		// Antenna info
		typeIcon := getTypeIcon(antenna.Config.Type)
		serialStr := fmt.Sprintf("%s %s [%s]", typeIcon, antenna.Config.Serial, antenna.Config.Type)

		// Cards with styling
		var cardsStr string
		if len(antenna.Cards) > 0 {
			cardsStr = cardStyle.Render(strings.Join(antenna.Cards, ", "))
		} else {
			cardsStr = inactiveStyle.Render("(no cards)")
		}

		// Status/Error
		var statusStr string
		if antenna.LastError != "" {
			statusStr = errorStyle.Render("✗ " + truncateString(antenna.LastError, 30))
		} else if antenna.LastStatus != "" {
			statusStr = successStyle.Render("✓ " + antenna.LastStatus)
		}

		// Build line
		line := fmt.Sprintf("  %s  %-30s  Cards: %-25s  %s",
			statusIndicator,
			serialStr,
			cardsStr,
			statusStr,
		)

		if isSelected {
			line = selectedStyle.Render(line)
		}

		b.WriteString(line)
		b.WriteString("\n")
	}

	// Input mode
	if m.inputMode == ModeAddCard {
		b.WriteString("\n")
		b.WriteString(inputStyle.Render("  Add card: "))
		b.WriteString(m.textInput.View())
		b.WriteString("\n")
	}

	// Message
	if m.lastMessage != "" {
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  %s\n", m.lastMessage))
	}

	// Help
	var help string
	if m.inputMode == ModeAddCard {
		help = helpStyle.Render("  enter: confirm • esc: cancel")
	} else {
		help = helpStyle.Render("  ↑/↓: navigate • enter/space: toggle • a: add card • d: delete • c: clear • q: quit")
	}
	b.WriteString("\n")
	b.WriteString(help)

	return b.String()
}

func getTypeIcon(antennaType string) string {
	switch antennaType {
	case "player":
		return "👤"
	case "board":
		return "🎴"
	case "muck":
		return "🗑️"
	default:
		return "❓"
	}
}

// getMaxCards returns the maximum number of cards for an antenna type
// Returns 0 for no limit
func getMaxCards(antennaType string) int {
	switch antennaType {
	case "player":
		return 2
	case "board":
		return 5
	default:
		return 0 // no limit for muck or unknown types
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// normalizeCardName formats card name with uppercase rank and lowercase suit
// e.g., "as" -> "As", "KH" -> "Kh", "10s" -> "Ts"
func normalizeCardName(card string) string {
	card = strings.TrimSpace(card)
	if len(card) < 2 {
		return card
	}

	// Handle the card - rank is first character(s), suit is last character
	suit := strings.ToLower(string(card[len(card)-1]))
	rank := strings.ToUpper(card[:len(card)-1])

	return rank + suit
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

func sendCard(client *http.Client, serverURL, uid, deviceID string, pairID int) error {
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
	httpReq.Header.Set("User-Agent", fmt.Sprintf("test-wired-client/%s (%s)", version.GetVersion(), version.GetCommit()))

	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated, http.StatusNotModified:
		return nil
	default:
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
	}
}
