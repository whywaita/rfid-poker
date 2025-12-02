-- Add card information (rank and suit) and player_name to hand_history table
ALTER TABLE hand_history ADD COLUMN `player_name` VARCHAR(255) NULL;
ALTER TABLE hand_history ADD COLUMN `card_a_rank` VARCHAR(255) NULL;
ALTER TABLE hand_history ADD COLUMN `card_a_suit` VARCHAR(255) NULL;
ALTER TABLE hand_history ADD COLUMN `card_b_rank` VARCHAR(255) NULL;
ALTER TABLE hand_history ADD COLUMN `card_b_suit` VARCHAR(255) NULL;
