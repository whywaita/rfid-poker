-- name: GetCard :one
SELECT id, suit, rank, hand_id, is_board FROM card WHERE id = ?;

-- name: GetCardByRankSuit :one
SELECT id, suit, rank, hand_id, is_board FROM card WHERE rank = ? AND suit = ?;

-- name: GetCardBySerial :many
SELECT id, suit, rank, hand_id, is_board FROM card WHERE serial = ?;

-- name: AddCard :one
INSERT INTO card (suit, rank, serial, is_board) VALUES (?, ?, ?, ?) RETURNING id;

-- name: SetCardHandByCardID :one
UPDATE card SET hand_id = ?
WHERE id = ?
RETURNING *;

-- name: DeleteBoardCards :exec
DELETE FROM card WHERE is_board = true;

-- name: DeleteCardAll :exec
DELETE FROM card;