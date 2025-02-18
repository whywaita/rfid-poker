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

-- name: AddHand :execresult
INSERT INTO hand (player_id, is_muck)
VALUES (?, false);

-- name: UpdateEquity :exec
UPDATE hand SET equity = ? WHERE id = ?;

-- name: ResetEquity :exec
UPDATE hand SET equity = 0;

-- name: MuckHand :exec
UPDATE hand SET is_muck = true WHERE id = ?;

-- name: DeleteHandAll :exec
DELETE FROM hand;