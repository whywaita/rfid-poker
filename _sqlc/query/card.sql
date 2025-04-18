-- name: GetCard :one
SELECT id, card_suit, card_rank, hand_id, is_board FROM card WHERE id = ?;

-- name: GetCardByRankSuit :one
SELECT id, card_suit, card_rank, hand_id, is_board FROM card WHERE card_rank = ? AND card_suit = ?;

-- name: GetCardBySerial :many
SELECT id, card_suit, card_rank, hand_id, is_board FROM card WHERE serial = ?;

-- name: AddCard :execresult
INSERT INTO card (card_suit, card_rank, serial, is_board) VALUES (?, ?, ?, ?);

-- name: SetCardHandByCardID :execresult
UPDATE card SET hand_id = ?
WHERE id = ?;

-- name: DeleteBoardCards :exec
DELETE FROM card WHERE is_board = true;

-- name: DeleteCardByAntennaID :exec
DELETE FROM card WHERE hand_id IN (SELECT id FROM hand WHERE player_id = (SELECT player_id FROM antenna WHERE antenna.id = ?));

-- name: DeleteCardAll :exec
DELETE FROM card;