package playercards

import (
	"encoding/hex"
	"fmt"
	"log"
	"strings"
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

	for _, v := range cached {
		if cards == v {
			// already cached
			return
		}
	}

	cache.Store(serial, append(cached, cards))
}

func ClearCache(serial string) {
	cache.Delete(serial)
}

func LoadCardsWithChannel(cc config.Config, ch chan HandData, sourceCh chan reader.Data) error {
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

		if len(LoadCache(in.SerialNumber)) == needNumber(cc, in.SerialNumber) {
			l := LoadCache(in.SerialNumber)
			log.Printf("loaded, will send to server: %v", l)
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

func needNumber(cc config.Config, serial string) int {
	if strings.EqualFold(cc.MuckSerial, serial) {
		return 2
	}
	if strings.EqualFold(cc.BoardSerial, serial) {
		return 3
	}

	return 2
}
