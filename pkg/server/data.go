package server

import (
	"log"
	"strings"

	"github.com/whywaita/poker-go"
	"github.com/whywaita/rfid-poker/pkg/config"
	"github.com/whywaita/rfid-poker/pkg/playercards"
)

func ReceiveData(ch chan playercards.HandData, updateCh chan struct{}, cc config.Config) error {
	for v := range ch {
		log.Printf("receive card in server: %s", v)
		if len(v.Cards) != 2 {
			log.Printf("invalid card: %s", v)
			continue
		}

		switch {
		case strings.EqualFold(v.SerialNumber, cc.MuckSerial):
			log.Printf("muck card: %s", v)
			MuckPlayer(poker.Player{
				Name: getPlayerName(v.SerialNumber, cc), // TODO: convert serial number to name
				Hand: v.Cards,
			})
		default:
			log.Printf("AddPlayer(): %s", v)
			_, err := AddPlayer(poker.Player{
				Name:  getPlayerName(v.SerialNumber, cc), // TODO: convert serial number to name
				Hand:  v.Cards,
				Score: 0,
			})
			if err != nil {
				log.Printf("Error AddPlayer(): %v", err)
				continue
			}
		}

		updateCh <- struct{}{}
	}

	return nil
}

func getPlayerName(serial string, cc config.Config) string {
	v, ok := cc.Players[serial]
	if !ok {
		return serial
	}
	return v
}
