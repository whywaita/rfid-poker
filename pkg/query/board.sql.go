// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: board.sql

package query

import (
	"context"
)

const addCardToBoard = `-- name: AddCardToBoard :exec
INSERT INTO card (suit, rank, is_board)
VALUES (?, ?, true)
`

type AddCardToBoardParams struct {
	Suit string
	Rank string
}

func (q *Queries) AddCardToBoard(ctx context.Context, arg AddCardToBoardParams) error {
	_, err := q.db.ExecContext(ctx, addCardToBoard, arg.Suit, arg.Rank)
	return err
}

const getBoard = `-- name: GetBoard :many
SELECT id, suit, rank, is_board FROM card
WHERE is_board = true
`

type GetBoardRow struct {
	ID      int64
	Suit    string
	Rank    string
	IsBoard bool
}

func (q *Queries) GetBoard(ctx context.Context) ([]GetBoardRow, error) {
	rows, err := q.db.QueryContext(ctx, getBoard)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetBoardRow
	for rows.Next() {
		var i GetBoardRow
		if err := rows.Scan(
			&i.ID,
			&i.Suit,
			&i.Rank,
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

const resetBoard = `-- name: ResetBoard :exec
DELETE FROM card
WHERE is_board = true
`

func (q *Queries) ResetBoard(ctx context.Context) error {
	_, err := q.db.ExecContext(ctx, resetBoard)
	return err
}
