package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"slices"
	"sort"
	"sync"

	"github.com/whywaita/poker-go"
	"github.com/whywaita/rfid-poker/pkg/query"
)

type Stored struct {
	PlayerName string
	Hand       []poker.Card
	Equity     float64
}

var (
	calcEquityMu sync.RWMutex
)

func calcEquity(ctx context.Context, q *query.Queries) error {
	calcEquityMu.Lock()
	defer calcEquityMu.Unlock()
	playersRow, err := q.GetPlayersWithHand(ctx)
	if err != nil {
		return fmt.Errorf("db.GetPlayersWithHand(): %w", err)
	}

	players := make([]poker.Player, 0, len(playersRow))

	for _, p := range playersRow {
		pgCardA, err := query.Card{
			Suit:    p.CardASuit,
			Rank:    p.CardARank,
			IsBoard: p.CardAIsBoard,
		}.ToPokerGo()
		if err != nil {
			return fmt.Errorf("cardA.ToPokerGo(): %w", err)
		}
		pgCardB, err := query.Card{
			Suit:    p.CardBSuit,
			Rank:    p.CardBRank,
			IsBoard: p.CardBIsBoard,
		}.ToPokerGo()
		if err != nil {
			return fmt.Errorf("cardB.ToPokerGo(): %w", err)
		}

		players = append(players, poker.Player{
			Name: p.Name,
			Hand: []poker.Card{
				*pgCardA,
				*pgCardB,
			},
		})
	}

	board, err := GetBoard(ctx, q)
	if err != nil {
		return fmt.Errorf("GetBoard(): %w", err)
	}

	log.Println("Start EvaluateEquityByMadeHandWithCommunity")
	equities, err := poker.EvaluateEquityByMadeHandWithCommunity(players, board)
	if err != nil {
		return fmt.Errorf("poker.EvaluateEquityByMadeHandWithCommunity: %w", err)
	}
	log.Println("End EvaluateEquityByMadeHand")

	for i, p := range playersRow {
		if err := q.UpdateEquity(ctx, query.UpdateEquityParams{
			Equity: sql.NullFloat64{
				Float64: equities[i],
				Valid:   true,
			},
			ID: p.HandID,
		}); err != nil {
			return fmt.Errorf("db.UpdatePlayerEquity(hand_id: %v): %w", p.HandID, err)
		}
	}

	return nil
}

func AddPlayer(ctx context.Context, conn *sql.DB, input []poker.Card, serial string) error {
	if len(input) != 2 {
		return fmt.Errorf("invalid input length (not 2): %v", input)
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("conn.BeginTx(): %w", err)
	}
	defer func() {
		if r := recover(); r != nil || err != nil {
			tx.Rollback()
		}
	}()
	qWithTx := query.New(tx)

	sort.SliceStable(input, func(i, j int) bool {
		return input[i].Rank < input[j].Rank
	})

	var storedCardID []int64
	for _, c := range input {
		isStored, err := isStoredCard(ctx, qWithTx, c)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("isStoredCard(): %w", err)
		}

		if isStored {
			tx.Rollback()
			return fmt.Errorf("card %v is already stored", c)
		}

		cardID, err := qWithTx.AddCard(ctx, query.AddCardParams{
			Suit: c.Suit.String(),
			Rank: c.Rank.String(),
		})
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("q.AddCard(): %w", err)
		}
		storedCardID = append(storedCardID, cardID)
	}
	player, err := qWithTx.GetPlayerBySerial(ctx, serial)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("q.GetPlayerBySerial(): %w", err)
	}

	if err := qWithTx.AddHand(ctx, query.AddHandParams{
		PlayerID: player.ID,
		CardAID:  storedCardID[0],
		CardBID:  storedCardID[1],
	}); err != nil {
		tx.Rollback()
		return fmt.Errorf("db.AddHand(): %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx.Commit(): %w", err)
	}

	if err := calcEquity(ctx, query.New(conn)); err != nil {
		return fmt.Errorf("calcEquity: %w", err)
	}

	return nil
}

func AddBoard(ctx context.Context, q *query.Queries, cards []poker.Card) error {
	nowBoard, err := GetBoard(ctx, q)
	if err != nil {
		return fmt.Errorf("GetBoard(): %w", err)
	}

	board, needInsert, isUpdated := concatCards(nowBoard, cards)
	if len(board) > 5 {
		return fmt.Errorf("concatenated length is %d, it is over five", len(board))
	}

	if len(needInsert) > 0 {
		for _, c := range needInsert {
			err := q.AddCardToBoard(ctx, query.AddCardToBoardParams{
				Suit: c.Suit.String(),
				Rank: c.Rank.String(),
			})
			if err != nil {
				return fmt.Errorf("query.AddCardToBoard(): %w", err)
			}
		}
	}

	if isUpdated {
		if err := calcEquity(ctx, q); err != nil {
			log.Printf("calcEquity: %v", err)
			return fmt.Errorf("calcEquity: %w", err)
		}
	}
	return nil
}

func isStoredCard(ctx context.Context, q *query.Queries, card poker.Card) (bool, error) {
	_, err := q.GetCardByRankSuit(ctx, query.GetCardByRankSuitParams{
		Rank: card.Rank.String(),
		Suit: card.Suit.String(),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("db.GetCardByRankSuit(): %w", err)
	}
	return true, nil
}

func MuckPlayer(ctx context.Context, q *query.Queries, cards []poker.Card) error {
	c, err := q.GetCardByRankSuit(ctx, query.GetCardByRankSuitParams{
		Rank: cards[0].Rank.String(),
		Suit: cards[0].Suit.String(),
	})
	if err != nil {
		return fmt.Errorf("q.GetCardByRankSuit(): %w", err)
	}
	hand, err := q.GetHandByCardId(ctx, query.GetHandByCardIdParams{
		CardAID: c.ID,
		CardBID: c.ID,
	})
	if err != nil {
		return fmt.Errorf("q.GetHandByCardId(): %w", err)
	}

	if err := q.MuckHand(ctx, hand.ID); err != nil {
		return fmt.Errorf("q.MuckHand(): %w", err)
	}

	if err := calcEquity(ctx, q); err != nil {
		return fmt.Errorf("calcEquity: %w", err)
	}

	return nil
}

func ClearGame(ctx context.Context, conn *sql.DB) error {
	db := query.New(conn)

	if err := db.ResetGame(ctx); err != nil {
		return fmt.Errorf("db.ResetGame(): %w", err)
	}

	return nil
}

func GetStored(ctx context.Context, q *query.Queries) ([]Stored, error) {
	players, err := q.GetPlayersWithHand(ctx)
	if err != nil {
		return nil, fmt.Errorf("q.GetPlayersWithHand(): %w", err)
	}
	stored := make([]Stored, 0, len(players))

	for _, p := range players {
		cardA := poker.Card{
			Suit: poker.UnmarshalSuitString(p.CardASuit),
			Rank: poker.UnmarshalRankString(p.CardARank),
		}
		cardB := poker.Card{
			Suit: poker.UnmarshalSuitString(p.CardBSuit),
			Rank: poker.UnmarshalRankString(p.CardBRank),
		}

		stored = append(stored, Stored{
			PlayerName: p.Name,
			Hand: []poker.Card{
				cardA,
				cardB,
			},
			Equity: p.Equity.Float64,
		})
	}

	return stored, nil
}

func GetBoard(ctx context.Context, q *query.Queries) ([]poker.Card, error) {
	cards, err := q.GetBoard(ctx)
	if err != nil {
		return nil, fmt.Errorf("db.GetBoard(): %w", err)
	}

	var board []poker.Card
	for _, c := range cards {
		card, err := c.ToPokerGo()
		if err != nil {
			return nil, fmt.Errorf("card.ToPokerGo(): %w", err)
		}
		board = append(board, *card)
	}

	return board, nil
}

// concatCards concat already stored cards and new cards (remove duplicated)
func concatCards(already, newCards []poker.Card) ([]poker.Card, []poker.Card, bool) {
	concat := make([]poker.Card, 0)
	concat = append(concat, already...)
	needInsert := make([]poker.Card, 0)

	isUpdated := false

	for _, newCard := range newCards {
		if slices.Contains(concat, newCard) {
			// already stored, ignore
			continue
		}
		concat = append(concat, newCard)
		needInsert = append(needInsert, newCard)
		isUpdated = true
	}

	return concat, needInsert, isUpdated
}
