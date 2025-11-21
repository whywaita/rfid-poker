-- Drop hand_history table
DROP TABLE hand_history;

-- Drop foreign keys and columns from card table
ALTER TABLE card DROP FOREIGN KEY fk_card_game;
ALTER TABLE card DROP COLUMN `game_id`;

-- Drop foreign keys and columns from hand table
ALTER TABLE hand DROP FOREIGN KEY fk_hand_game;
ALTER TABLE hand DROP COLUMN `game_id`;

-- Drop game table
DROP TABLE game;
