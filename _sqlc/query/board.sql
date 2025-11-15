-- name: GetBoard :many
SELECT id, card_suit, card_rank, serial, is_board FROM card
WHERE is_board = true;

-- name: AddCardToBoard :exec
INSERT INTO card (card_suit, card_rank, serial, is_board, game_id)
VALUES (?, ?, ?, true, ?);

-- name: ResetBoard :exec
DELETE FROM card
WHERE is_board = true;