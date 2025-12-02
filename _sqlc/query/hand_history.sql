-- name: CopyHandsToHistory :exec
INSERT INTO hand_history (game_id, player_id, player_name, equity, is_muck, card_a_rank, card_a_suit, card_b_rank, card_b_suit)
SELECT hand.game_id, hand.player_id, player.name, hand.equity, hand.is_muck,
       card_a.card_rank, card_a.card_suit,
       card_b.card_rank, card_b.card_suit
FROM hand
JOIN player ON hand.player_id = player.id
LEFT JOIN card AS card_a ON hand.id = card_a.hand_id
LEFT JOIN card AS card_b ON hand.id = card_b.hand_id AND card_a.id < card_b.id
WHERE hand.game_id = ?
  AND (card_b.id IS NOT NULL OR card_a.id IS NULL);

-- name: GetHandHistoryByGameID :many
SELECT id, game_id, player_id, player_name, equity, is_muck, card_a_rank, card_a_suit, card_b_rank, card_b_suit, created_at
FROM hand_history
WHERE game_id = ?
ORDER BY created_at DESC;

-- name: GetHandHistoryByPlayerID :many
SELECT id, game_id, player_id, player_name, equity, is_muck, card_a_rank, card_a_suit, card_b_rank, card_b_suit, created_at
FROM hand_history
WHERE player_id = ?
ORDER BY created_at DESC;
