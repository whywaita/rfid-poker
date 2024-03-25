// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.25.0
// source: hand.sql

package query

import (
	"context"
	"database/sql"
)

const addHand = `-- name: AddHand :exec
INSERT INTO hand (player_id, card_a_id, card_b_id, is_muck) VALUES (?, ?, ?, false)
`

type AddHandParams struct {
	PlayerID interface{}
	CardAID  interface{}
	CardBID  interface{}
}

func (q *Queries) AddHand(ctx context.Context, arg AddHandParams) error {
	_, err := q.db.ExecContext(ctx, addHand, arg.PlayerID, arg.CardAID, arg.CardBID)
	return err
}

const deleteHandAll = `-- name: DeleteHandAll :exec
DELETE FROM hand
`

func (q *Queries) DeleteHandAll(ctx context.Context) error {
	_, err := q.db.ExecContext(ctx, deleteHandAll)
	return err
}

const getHandByCardId = `-- name: GetHandByCardId :one
SELECT id, player_id, card_a_id, card_b_id, equity FROM hand WHERE card_a_id = ? OR card_b_id = ?
`

type GetHandByCardIdParams struct {
	CardAID interface{}
	CardBID interface{}
}

type GetHandByCardIdRow struct {
	ID       int64
	PlayerID interface{}
	CardAID  interface{}
	CardBID  interface{}
	Equity   sql.NullFloat64
}

func (q *Queries) GetHandByCardId(ctx context.Context, arg GetHandByCardIdParams) (GetHandByCardIdRow, error) {
	row := q.db.QueryRowContext(ctx, getHandByCardId, arg.CardAID, arg.CardBID)
	var i GetHandByCardIdRow
	err := row.Scan(
		&i.ID,
		&i.PlayerID,
		&i.CardAID,
		&i.CardBID,
		&i.Equity,
	)
	return i, err
}

const getHandBySerial = `-- name: GetHandBySerial :one
SELECT
    hand.id AS hand_id,
    player_id,
    card_a_id,
    card_b_id,
    equity
FROM hand JOIN player ON player.id = hand.player_id
WHERE player.serial = ?
`

type GetHandBySerialRow struct {
	HandID   int64
	PlayerID interface{}
	CardAID  interface{}
	CardBID  interface{}
	Equity   sql.NullFloat64
}

func (q *Queries) GetHandBySerial(ctx context.Context, serial string) (GetHandBySerialRow, error) {
	row := q.db.QueryRowContext(ctx, getHandBySerial, serial)
	var i GetHandBySerialRow
	err := row.Scan(
		&i.HandID,
		&i.PlayerID,
		&i.CardAID,
		&i.CardBID,
		&i.Equity,
	)
	return i, err
}

const getHandNotMucked = `-- name: GetHandNotMucked :many
SELECT id, player_id, card_a_id, card_b_id, equity FROM hand WHERE is_muck = false
`

type GetHandNotMuckedRow struct {
	ID       int64
	PlayerID interface{}
	CardAID  interface{}
	CardBID  interface{}
	Equity   sql.NullFloat64
}

func (q *Queries) GetHandNotMucked(ctx context.Context) ([]GetHandNotMuckedRow, error) {
	rows, err := q.db.QueryContext(ctx, getHandNotMucked)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetHandNotMuckedRow
	for rows.Next() {
		var i GetHandNotMuckedRow
		if err := rows.Scan(
			&i.ID,
			&i.PlayerID,
			&i.CardAID,
			&i.CardBID,
			&i.Equity,
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

const muckHand = `-- name: MuckHand :exec
UPDATE hand SET is_muck = true WHERE id = ?
`

func (q *Queries) MuckHand(ctx context.Context, id int64) error {
	_, err := q.db.ExecContext(ctx, muckHand, id)
	return err
}

const updateEquity = `-- name: UpdateEquity :exec
UPDATE hand SET equity = ? WHERE id = ?
`

type UpdateEquityParams struct {
	Equity sql.NullFloat64
	ID     int64
}

func (q *Queries) UpdateEquity(ctx context.Context, arg UpdateEquityParams) error {
	_, err := q.db.ExecContext(ctx, updateEquity, arg.Equity, arg.ID)
	return err
}
