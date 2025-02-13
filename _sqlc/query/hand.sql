-- name: GetHand :one
SELECT id, player_id, equity FROM hand WHERE id = ? LIMIT 1;

-- name: GetHandBySerial :one
SELECT
    hand.id AS hand_id,
    antenna.player_id,
    equity
FROM hand JOIN antenna ON antenna.player_id = hand.player_id
WHERE antenna.serial = ?;

-- name: GetHandNotMucked :many
SELECT id, player_id, equity FROM hand WHERE is_muck = false;

-- name: AddHand :one
INSERT INTO hand (player_id, is_muck)
VALUES (?, false)
RETURNING *;

-- name: UpdateEquity :exec
UPDATE hand SET equity = ? WHERE id = ?;

-- name: MuckHand :exec
UPDATE hand SET is_muck = true WHERE id = ?;

-- name: DeleteHandAll :exec
DELETE FROM hand;