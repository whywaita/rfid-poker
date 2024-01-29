-- name: GetBoard :many
SELECT * FROM card
WHERE is_board = true;

-- name: AddCardToBoard :exec
INSERT INTO card (suit, rank, is_board)
VALUES (?, ?, true);

-- name: ResetBoard :exec
DELETE FROM card
WHERE is_board = true;