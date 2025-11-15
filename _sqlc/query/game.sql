-- name: CreateGame :exec
INSERT INTO game (id, status)
VALUES (?, 'active');

-- name: GetCurrentGame :one
SELECT id, started_at, ended_at, status FROM game WHERE status = 'active' ORDER BY started_at DESC LIMIT 1;

-- name: GetGameByID :one
SELECT id, started_at, ended_at, status FROM game WHERE id = ? LIMIT 1;

-- name: FinishGame :exec
UPDATE game SET ended_at = NOW(), status = 'finished' WHERE id = ?;

-- name: DeleteAllGames :exec
DELETE FROM game;
