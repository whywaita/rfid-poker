-- name: GetHand :one
SELECT id, player_id, equity FROM hand WHERE id = ? LIMIT 1;

-- name: GetHandBySerial :one
SELECT
    hand.id AS hand_id,
    antenna.player_id,
    equity
FROM hand JOIN antenna ON antenna.player_id = hand.player_id
WHERE antenna.serial = ?;

-- name: GetHandWithCardByPlayerID :one
SELECT
    hand.id AS hand_id,
    hand.player_id,
    hand.is_muck,
    equity,
    card_a.card_suit AS card_a_suit,
    card_a.card_rank AS card_a_rank,
    card_a.is_board AS card_a_is_board,
    card_b.card_suit AS card_b_suit,
    card_b.card_rank AS card_b_rank,
    card_b.is_board AS card_b_is_board
FROM hand
         JOIN card AS card_a ON hand.id = card_a.hand_id
         JOIN card AS card_b ON hand.id = card_b.hand_id
WHERE hand.player_id = ?
  AND card_a.id < card_b.id;

-- name: GetHandNotMucked :many
SELECT id, player_id, equity FROM hand WHERE is_muck = false;

-- name: AddHand :execresult
INSERT INTO hand (player_id, is_muck, game_id)
VALUES (?, false, ?);

-- name: UpdateEquity :exec
UPDATE hand SET equity = ? WHERE id = ?;

-- name: ResetEquity :exec
UPDATE hand SET equity = 0;

-- name: MuckHand :exec
UPDATE hand SET is_muck = true WHERE id = ?;

-- name: DeleteHandByAntennaID :exec
DELETE FROM hand WHERE player_id = (SELECT player_id FROM antenna WHERE antenna.id = ?);

-- name: DeleteHandAll :exec
DELETE FROM hand;

-- name: DeleteHandByGameID :exec
DELETE FROM hand WHERE game_id = ?;