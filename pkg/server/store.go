package server

import (
	"fmt"
	"log"
	"slices"
	"sort"

	"github.com/whywaita/poker-go"
)

var (
	storedPlayers = make([]Stored, 0)
	storedBoard   = make([]poker.Card, 0)
)

type Stored struct {
	Player poker.Player
	Equity float64
}

func calcEquity(stored []Stored) ([]Stored, error) {
	if len(stored) <= 1 {
		return stored, nil
	}
	players := make([]poker.Player, len(stored))
	for i, s := range stored {
		players[i] = s.Player
	}

	board := GetBoard()

	log.Println("Start EvaluateEquityByMadeHandWithCommunity")
	equities, err := poker.EvaluateEquityByMadeHandWithCommunity(players, board)
	if err != nil {
		return nil, fmt.Errorf("poker.EvaluateEquityByMadeHandWithCommunity: %w", err)
	}
	log.Println("End EvaluateEquityByMadeHand")

	s := make([]Stored, len(stored))
	for i := range stored {
		s[i] = Stored{
			Player: players[i],
			Equity: equities[i],
		}
	}

	return s, nil
}

func AddPlayer(p poker.Player) ([]Stored, error) {
	sort.SliceStable(p.Hand, func(i, j int) bool {
		return p.Hand[i].Rank < p.Hand[j].Rank
	})

	for _, c := range p.Hand {
		if isStoredCard(c) {
			return nil, fmt.Errorf("card %v is already stored", c)
		}
	}

	if len(storedPlayers) == 0 {
		storedPlayers = []Stored{
			{
				Player: p,
				Equity: 0,
			},
		}
	} else {
		updated := false
		for i, stored := range storedPlayers {
			if p.Name == stored.Player.Name {
				log.Println("AddPlayer: Update", p.Name)

				storedPlayers[i].Player = p
				updated = true
				break
			}
		}
		if !updated {
			storedPlayers = append(storedPlayers, Stored{
				Player: p,
				Equity: 0,
			})
		}
	}

	stored, err := calcEquity(storedPlayers)
	if err != nil {
		return nil, fmt.Errorf("calcEquity: %w", err)
	}
	storedPlayers = stored

	return stored, nil
}

func AddBoard(cards []poker.Card) error {
	board, isUpdated := concatCards(GetBoard(), cards)
	if len(board) > 5 {
		return fmt.Errorf("board is already 5 cards")
	}

	storedBoard = board

	if isUpdated {
		stored, err := calcEquity(storedPlayers)
		if err != nil {
			log.Printf("calcEquity: %v", err)
			return fmt.Errorf("calcEquity: %w", err)
		}
		storedPlayers = stored
	}
	return nil
}

func isStoredCard(card poker.Card) bool {
	for _, stored := range storedPlayers {
		for _, storedCard := range stored.Player.Hand {
			if storedCard == card {
				return true
			}
		}
	}

	return false
}

func MuckPlayer(p poker.Player) {
	for i, v := range storedPlayers {
		if v.Player.Name == p.Name {
			storedPlayers = append(storedPlayers[:i], storedPlayers[i+1:]...)
		}
	}
}

func ClearGame() {
	storedPlayers = nil
	storedBoard = nil
}

func GetStored() []Stored {
	return storedPlayers
}

func GetBoard() []poker.Card {
	return storedBoard
}

// concatCards concat already stored cards and new cards (remove duplicated)
func concatCards(already, newCards []poker.Card) ([]poker.Card, bool) {
	concat := make([]poker.Card, 0)
	concat = append(concat, already...)

	isUpdated := false

	for _, newCard := range newCards {
		if slices.Contains(concat, newCard) {
			continue
		}
		concat = append(concat, newCard)
		isUpdated = true
	}

	return concat, isUpdated
}
