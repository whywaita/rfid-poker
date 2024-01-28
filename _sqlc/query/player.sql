-- name: GetPlayer :one
SELECT * FROM player
WHERE id = ? LIMIT 1;

-- name: GetPlayerBySerial :one
SELECT * FROM player
WHERE serial = ? LIMIT 1;

-- name: GetPlayersWithHand :many
SELECT
    player.id,
    player.name,
    player.serial,
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
         INNER JOIN card AS card_a ON hand.card_a_id = card_a.id
         INNER JOIN card AS card_b ON hand.card_b_id = card_b.id;

-- name: AddPlayer :one
INSERT INTO player (name, serial)
VALUES (?, ?)
RETURNING *;