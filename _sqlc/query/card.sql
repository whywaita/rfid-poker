-- name: GetCard :one
SELECT id, suit, rank, is_board FROM card WHERE id = ?;

-- name: GetCardByRankSuit :one
SELECT id, suit, rank, is_board FROM card WHERE rank = ? AND suit = ?;

-- name: AddCard :one
INSERT INTO card (suit, rank, is_board) VALUES (?, ?, ?) RETURNING id;