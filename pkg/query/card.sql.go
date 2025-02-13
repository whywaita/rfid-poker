// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: card.sql

package query

import (
	"context"
	"database/sql"
)

const addCard = `-- name: AddCard :one
INSERT INTO card (suit, rank, serial, is_board) VALUES (?, ?, ?, ?) RETURNING id
`

type AddCardParams struct {
	Suit    string
	Rank    string
	Serial  string
	IsBoard bool
}

func (q *Queries) AddCard(ctx context.Context, arg AddCardParams) (int64, error) {
	row := q.db.QueryRowContext(ctx, addCard,
		arg.Suit,
		arg.Rank,
		arg.Serial,
		arg.IsBoard,
	)
	var id int64
	err := row.Scan(&id)
	return id, err
}

const deleteCardAll = `-- name: DeleteCardAll :exec
DELETE FROM card
`

func (q *Queries) DeleteCardAll(ctx context.Context) error {
	_, err := q.db.ExecContext(ctx, deleteCardAll)
	return err
}

const getCard = `-- name: GetCard :one
SELECT id, suit, rank, hand_id, is_board FROM card WHERE id = ?
`

type GetCardRow struct {
	ID      int64
	Suit    string
	Rank    string
	HandID  sql.NullInt64
	IsBoard bool
}

func (q *Queries) GetCard(ctx context.Context, id int64) (GetCardRow, error) {
	row := q.db.QueryRowContext(ctx, getCard, id)
	var i GetCardRow
	err := row.Scan(
		&i.ID,
		&i.Suit,
		&i.Rank,
		&i.HandID,
		&i.IsBoard,
	)
	return i, err
}

const getCardByRankSuit = `-- name: GetCardByRankSuit :one
SELECT id, suit, rank, hand_id, is_board FROM card WHERE rank = ? AND suit = ?
`

type GetCardByRankSuitParams struct {
	Rank string
	Suit string
}

type GetCardByRankSuitRow struct {
	ID      int64
	Suit    string
	Rank    string
	HandID  sql.NullInt64
	IsBoard bool
}

func (q *Queries) GetCardByRankSuit(ctx context.Context, arg GetCardByRankSuitParams) (GetCardByRankSuitRow, error) {
	row := q.db.QueryRowContext(ctx, getCardByRankSuit, arg.Rank, arg.Suit)
	var i GetCardByRankSuitRow
	err := row.Scan(
		&i.ID,
		&i.Suit,
		&i.Rank,
		&i.HandID,
		&i.IsBoard,
	)
	return i, err
}

const getCardBySerial = `-- name: GetCardBySerial :many
SELECT id, suit, rank, hand_id, is_board FROM card WHERE serial = ?
`

type GetCardBySerialRow struct {
	ID      int64
	Suit    string
	Rank    string
	HandID  sql.NullInt64
	IsBoard bool
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
			&i.Suit,
			&i.Rank,
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

const setCardHandByCardID = `-- name: SetCardHandByCardID :one
UPDATE card SET hand_id = ?
WHERE id = ?
RETURNING id, suit, rank, is_board, hand_id, serial
`

type SetCardHandByCardIDParams struct {
	HandID sql.NullInt64
	ID     int64
}

func (q *Queries) SetCardHandByCardID(ctx context.Context, arg SetCardHandByCardIDParams) (Card, error) {
	row := q.db.QueryRowContext(ctx, setCardHandByCardID, arg.HandID, arg.ID)
	var i Card
	err := row.Scan(
		&i.ID,
		&i.Suit,
		&i.Rank,
		&i.IsBoard,
		&i.HandID,
		&i.Serial,
	)
	return i, err
}
