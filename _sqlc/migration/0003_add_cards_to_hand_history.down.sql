-- Remove card information and player_name from hand_history table
ALTER TABLE hand_history DROP COLUMN `card_b_suit`;
ALTER TABLE hand_history DROP COLUMN `card_b_rank`;
ALTER TABLE hand_history DROP COLUMN `card_a_suit`;
ALTER TABLE hand_history DROP COLUMN `card_a_rank`;
ALTER TABLE hand_history DROP COLUMN `player_name`;
