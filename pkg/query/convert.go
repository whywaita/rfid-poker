package query

import (
	"fmt"

	"github.com/whywaita/poker-go"
)

func (c Card) ToPokerGo() (*poker.Card, error) {
	r := poker.UnmarshalRankString(c.CardRank)
	if r == poker.RankUnknown {
		return nil, fmt.Errorf("unknown rank: %s", c.CardRank)
	}
	s := poker.UnmarshalSuitString(c.CardSuit)
	if s == -1 {
		return nil, fmt.Errorf("unknown suit: %s", c.CardSuit)
	}

	return &poker.Card{
		Rank: r,
		Suit: s,
	}, nil
}
