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
	for in := range sourceCh {
		in := in
		go func() {
			card, err := LoadPlayerCard(in.UID, cc.CardIDs)
			if err != nil {
				log.Printf("playercards.LoadPlayerCard(%s, cardConfigs): %v", hex.EncodeToString(in.UID), err)
				return
			}

			cached := LoadCache(in.SerialNumber)
			switch getTypeSerial(cc, in.SerialNumber) {
			case TypeSerialBoard:
				// Send cache anyway if board
				SaveCache(in.SerialNumber, card)
				if err := sendCache(in.SerialNumber, ch); err != nil {
					log.Printf("sendCache(%s): %v", in.SerialNumber, err)
					return
				}
			default:
				switch len(cached) {
				case 0:
					log.Printf("cache.Store(%s, %s)", in.SerialNumber, card)
					cache.Store(in.SerialNumber, []string{card})
					return
				default:
					// add card if not cached
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
				}
			}

			// send cache if already reach correct number in cached
			if isCorrectNumber(cc, in.SerialNumber, LoadCache(in.SerialNumber)) {
				if err := sendCache(in.SerialNumber, ch); err != nil {
					log.Printf("sendCache(%s): %v", in.SerialNumber, err)
					return
				}
			}
		}()
	}

	return nil
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

type TypeSerial int

const (
	TypeSerialUnknown = iota
	TypeSerialMuck
	TypeSerialBoard
	TypeSerialPlayer
)

func (t TypeSerial) String() string {
	switch t {
	case TypeSerialMuck:
		return "muck"
	case TypeSerialBoard:
		return "board"
	case TypeSerialPlayer:
		return "player"
	default:
		return "unknown"
	}
}

func getTypeSerial(cc config.Config, serial string) TypeSerial {
	if strings.EqualFold(cc.MuckSerial, serial) {
		return TypeSerialMuck
	}
	if strings.EqualFold(cc.BoardSerial, serial) {
		return TypeSerialBoard
	}

	return TypeSerialPlayer
}

func isCorrectNumber(cc config.Config, serial string, cards []string) bool {
	t := getTypeSerial(cc, serial)
	if t == TypeSerialBoard {
		return len(cards) >= 1
	}

	return len(cards) == getCorrectNumber(cc, serial)
}

func getCorrectNumber(cc config.Config, serial string) int {
	t := getTypeSerial(cc, serial)
	switch t {
	case TypeSerialMuck:
		return 2
	case TypeSerialBoard:
		return 1
	case TypeSerialPlayer:
		return 2
	default:
		return -1
	}
}

func sendCache(serial string, ch chan HandData) error {
	l := LoadCache(serial)
	log.Printf("loaded, will send to server: %v", l)
	cards, err := ValidateCards(l)
	if err != nil {
		return fmt.Errorf("playercards.ValidateCards(cc, loaded): %w", err)
	}
	ch <- HandData{
		Cards:        cards,
		SerialNumber: serial,
	}
	ClearCache(serial)
	already = append(already, l...)
	return nil
}
