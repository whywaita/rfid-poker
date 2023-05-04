package playercards

import (
	"encoding/hex"
	"fmt"
	"log"
	"sync"

	"github.com/whywaita/poker-go"
	"github.com/whywaita/rfid-poker/pkg/config"
	"github.com/whywaita/rfid-poker/pkg/reader"
)

type HandData struct {
	Cards        []poker.Card
	SerialNumber string
}

var already []string
var cache sync.Map

func LoadCache(serial string) []string {
	loaded, ok := cache.Load(serial)
	if !ok {
		return nil
	}
	s, ok := loaded.([]string)
	if !ok {
		return nil
	}
	return s
}

func SaveCache(serial string, cards string) {
	cached := LoadCache(serial)
	if cached == nil {
		cache.Store(serial, []string{cards})
		return
	}

	cache.Store(serial, append(cached, cards))
}

func ClearCache(serial string) {
	cache.Delete(serial)
}

func LoadCardsWithChannel(cc config.Config, number int, ch chan HandData, sourceCh chan reader.Data) error {
	for {
		in := <-sourceCh

		card, err := LoadPlayerCard(in.UID, cc.CardIDs)
		if err != nil {
			log.Printf("playercards.LoadPlayerCard(%s, cardConfigs): %v", hex.EncodeToString(in.UID), err)
			continue
		}

		cached := LoadCache(in.SerialNumber)
		if len(cached) == 0 {
			cache.Store(in.SerialNumber, []string{card})
			continue
		}

		for _, v := range cached {
			if card != v {
				for _, a := range already {
					if card == a {
						continue
					}
				}

				log.Printf("found: %s", card)
				SaveCache(in.SerialNumber, card)
				continue
			} else {
				continue
			}
		}

		if len(LoadCache(in.SerialNumber)) == number {
			l := LoadCache(in.SerialNumber)
			log.Printf("loaded: %v", l)
			cards, err := ValidateCards(l)
			if err != nil {
				log.Printf("playercards.ValidateCards(cc, loaded): %v", err)
				continue
			}
			ch <- HandData{
				Cards:        cards,
				SerialNumber: in.SerialNumber,
			}
			ClearCache(in.SerialNumber)
			already = append(already, l...)
			continue
		}
	}
}

func ValidateCards(cards []string) ([]poker.Card, error) {
	validated := make([]poker.Card, 0, len(cards))

	for _, card := range cards {
		v, err := UnmarshalPlayerCard(card)
		if err != nil {
			return nil, fmt.Errorf("playercards.UnmarshalPlayerCard(%s): %w", card, err)
		}
		validated = append(validated, v)
	}

	return validated, nil
}
