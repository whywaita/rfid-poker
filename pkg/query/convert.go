package query

import (
	"fmt"

	"github.com/whywaita/poker-go"
)

func (c Card) ToPokerGo() (*poker.Card, error) {
	r := poker.UnmarshalRankString(c.Rank)
	if r == poker.RankUnknown {
		return nil, fmt.Errorf("unknown rank: %s", c.Rank)
	}
	s := poker.UnmarshalSuitString(c.Suit)
	if s == -1 {
		return nil, fmt.Errorf("unknown suit: %s", c.Suit)
	}

	return &poker.Card{
		Rank: r,
		Suit: s,
	}, nil
}
