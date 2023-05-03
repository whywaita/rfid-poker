package playercards

import (
	"fmt"

	"github.com/whywaita/poker-go"
)

func UnmarshalPlayerCard(in string) (poker.Card, error) {
	var card poker.Card

	if len(in) != 2 {
		return card, fmt.Errorf("invalid card length: %s", in)
	}

	switch in[1] {
	case 's':
		card.Suit = poker.Spades
	case 'h':
		card.Suit = poker.Hearts
	case 'd':
		card.Suit = poker.Diamonds
	case 'c':
		card.Suit = poker.Clubs
	default:
		return card, fmt.Errorf("invalid suit: %s", in)
	}

	switch in[0] {
	case 'A':
		card.Rank = poker.RankAce
	case 'T':
		card.Rank = poker.RankTen
	case 'J':
		card.Rank = poker.RankJack
	case 'Q':
		card.Rank = poker.RankQueen
	case 'K':
		card.Rank = poker.RankKing
	default:
		rank := poker.UnmarshalRankString(in[0:1])
		if rank == poker.RankUnknown {
			return card, fmt.Errorf("invalid rank: %s", in)
		} else {
			card.Rank = rank
		}
	}

	return card, nil
}
