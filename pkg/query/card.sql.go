// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: card.sql

package query

import (
	"context"
	"database/sql"
)

const addCard = `-- name: AddCard :execresult
INSERT INTO card (card_suit, card_rank, serial, is_board) VALUES (?, ?, ?, ?)
`

type AddCardParams struct {
	CardSuit string
	CardRank string
	Serial   string
	IsBoard  bool
}

func (q *Queries) AddCard(ctx context.Context, arg AddCardParams) (sql.Result, error) {
	return q.db.ExecContext(ctx, addCard,
		arg.CardSuit,
		arg.CardRank,
		arg.Serial,
		arg.IsBoard,
	)
}

const deleteBoardCards = `-- name: DeleteBoardCards :exec
DELETE FROM card WHERE is_board = true
`

func (q *Queries) DeleteBoardCards(ctx context.Context) error {
	_, err := q.db.ExecContext(ctx, deleteBoardCards)
	return err
}

const deleteCardAll = `-- name: DeleteCardAll :exec
DELETE FROM card
`

func (q *Queries) DeleteCardAll(ctx context.Context) error {
	_, err := q.db.ExecContext(ctx, deleteCardAll)
	return err
}

const deleteCardByAntennaID = `-- name: DeleteCardByAntennaID :exec
DELETE FROM card WHERE hand_id IN (SELECT id FROM hand WHERE player_id = (SELECT player_id FROM antenna WHERE antenna.id = ?))
`

func (q *Queries) DeleteCardByAntennaID(ctx context.Context, id int32) error {
	_, err := q.db.ExecContext(ctx, deleteCardByAntennaID, id)
	return err
}

const getCard = `-- name: GetCard :one
SELECT id, card_suit, card_rank, hand_id, is_board FROM card WHERE id = ?
`

type GetCardRow struct {
	ID       int32
	CardSuit string
	CardRank string
	HandID   sql.NullInt32
	IsBoard  bool
}

func (q *Queries) GetCard(ctx context.Context, id int32) (GetCardRow, error) {
	row := q.db.QueryRowContext(ctx, getCard, id)
	var i GetCardRow
	err := row.Scan(
		&i.ID,
		&i.CardSuit,
		&i.CardRank,
		&i.HandID,
		&i.IsBoard,
	)
	return i, err
}

const getCardByRankSuit = `-- name: GetCardByRankSuit :one
SELECT id, card_suit, card_rank, hand_id, is_board FROM card WHERE card_rank = ? AND card_suit = ?
`

type GetCardByRankSuitParams struct {
	CardRank string
	CardSuit string
}

type GetCardByRankSuitRow struct {
	ID       int32
	CardSuit string
	CardRank string
	HandID   sql.NullInt32
	IsBoard  bool
}

func (q *Queries) GetCardByRankSuit(ctx context.Context, arg GetCardByRankSuitParams) (GetCardByRankSuitRow, error) {
	row := q.db.QueryRowContext(ctx, getCardByRankSuit, arg.CardRank, arg.CardSuit)
	var i GetCardByRankSuitRow
	err := row.Scan(
		&i.ID,
		&i.CardSuit,
		&i.CardRank,
		&i.HandID,
		&i.IsBoard,
	)
	return i, err
}

const getCardBySerial = `-- name: GetCardBySerial :many
SELECT id, card_suit, card_rank, hand_id, is_board FROM card WHERE serial = ?
`

type GetCardBySerialRow struct {
	ID       int32
	CardSuit string
	CardRank string
	HandID   sql.NullInt32
	IsBoard  bool
}

func (q *Queries) GetCardBySerial(ctx context.Context, serial string) ([]GetCardBySerialRow, error) {
	rows, err := q.db.QueryContext(ctx, getCardBySerial, serial)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetCardBySerialRow
	for rows.Next() {
		var i GetCardBySerialRow
		if err := rows.Scan(
			&i.ID,
			&i.CardSuit,
			&i.CardRank,
			&i.HandID,
			&i.IsBoard,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const setCardHandByCardID = `-- name: SetCardHandByCardID :execresult
UPDATE card SET hand_id = ?
WHERE id = ?
`

type SetCardHandByCardIDParams struct {
	HandID sql.NullInt32
	ID     int32
}

func (q *Queries) SetCardHandByCardID(ctx context.Context, arg SetCardHandByCardIDParams) (sql.Result, error) {
	return q.db.ExecContext(ctx, setCardHandByCardID, arg.HandID, arg.ID)
}
