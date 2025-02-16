-- name: GetPlayer :one
SELECT id, name FROM player
WHERE id = ? LIMIT 1;

-- name: GetPlayerBySerial :one
SELECT player.id, player.name
FROM player
JOIN antenna ON player.id = antenna.player_id
WHERE antenna.serial = ?;

-- name: GetPlayersWithHand :many
SELECT
    player.id,
    player.name,
    hand.id AS hand_id,
    hand.equity,
    hand.is_muck,
    card_a.suit AS card_a_suit,
    card_a.rank AS card_a_rank,
    card_a.is_board AS card_a_is_board,
    card_b.suit AS card_b_suit,
    card_b.rank AS card_b_rank,
    card_b.is_board AS card_b_is_board
FROM player
         INNER JOIN hand ON player.id = hand.player_id
         INNER JOIN card AS card_a ON hand.id = card_a.hand_id
         INNER JOIN card AS card_b ON hand.id = card_b.hand_id
WHERE hand.is_muck = false
  AND card_a.id < card_b.id;

-- name: AddPlayer :one
INSERT INTO player (name)
VALUES (?)
RETURNING *;

-- name: DeletePlayerWithHandWithCards :exec
DELETE FROM card
WHERE hand_id IN (SELECT id FROM hand WHERE player_id = ?);

DELETE FROM hand
WHERE player_id = ?;

DELETE FROM player
WHERE id = ?;