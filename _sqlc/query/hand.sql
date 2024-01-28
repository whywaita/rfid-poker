-- name: GetHandBySerial :one
SELECT
    hand.id AS hand_id,
    player_id,
    card_a_id,
    card_b_id,
    equity
FROM hand JOIN player ON player.id = hand.player_id
WHERE player.serial = ?;

-- name: GetHandNotMucked :many
SELECT id, player_id, card_a_id, card_b_id, equity FROM hand WHERE is_muck = false;

-- name: GetHandByCardId :one
SELECT id, player_id, card_a_id, card_b_id, equity FROM hand WHERE card_a_id = ? OR card_b_id = ?;

-- name: AddHand :exec
INSERT INTO hand (player_id, card_a_id, card_b_id, is_muck) VALUES (?, ?, ?, false);

-- name: UpdateEquity :exec
UPDATE hand SET equity = ? WHERE id = ?;

-- name: MuckHand :exec
UPDATE hand SET is_muck = true WHERE id = ?;