-- Drop foreign keys and columns from card table
ALTER TABLE card DROP FOREIGN KEY card_ibfk_2;
ALTER TABLE card DROP COLUMN `game_id`;

-- Drop foreign keys and columns from hand table
ALTER TABLE hand DROP FOREIGN KEY hand_ibfk_1;
ALTER TABLE hand DROP COLUMN `game_id`;

-- Drop game table
DROP TABLE game;
