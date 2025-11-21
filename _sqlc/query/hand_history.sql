-- name: CopyHandsToHistory :exec
INSERT INTO hand_history (game_id, player_id, equity, is_muck)
SELECT hand.game_id, hand.player_id, hand.equity, hand.is_muck
FROM hand
WHERE hand.game_id = ?;

-- name: GetHandHistoryByGameID :many
SELECT id, game_id, player_id, equity, is_muck, created_at
FROM hand_history
WHERE game_id = ?
ORDER BY created_at DESC;

-- name: GetHandHistoryByPlayerID :many
SELECT id, game_id, player_id, equity, is_muck, created_at
FROM hand_history
WHERE player_id = ?
ORDER BY created_at DESC;
