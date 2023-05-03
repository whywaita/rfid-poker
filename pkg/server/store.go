package server

import (
	"fmt"
	"log"
	"sort"

	"github.com/whywaita/poker-go"
)

var (
	storedPlayers = make([]Stored, 0)
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
	needUpdate := false
	for i, s := range stored {
		if s.Equity == 0 {
			needUpdate = true
		}
		players[i] = s.Player
	}
	if !needUpdate {
		return stored, nil
	}

	log.Println("Start EvaluateEquityByMadeHand")
	equities, err := poker.EvaluateEquityByMadeHand(players)
	if err != nil {
		return nil, fmt.Errorf("poker.EvaluateEquityByMadeHand: %w", err)
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
	log.Println("AddPlayer", p)
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

func ClearPlayers() {
	storedPlayers = nil
}

func MuckPlayer(p poker.Player) {
	for i, v := range storedPlayers {
		if v.Player.Name == p.Name {
			storedPlayers = append(storedPlayers[:i], storedPlayers[i+1:]...)
		}
	}
}

func GetStored() []Stored {
	return storedPlayers
}
