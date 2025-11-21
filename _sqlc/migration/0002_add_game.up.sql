-- Add game table for tracking individual games with UUID
CREATE TABLE game (
    `id` VARCHAR(36) PRIMARY KEY,
    `started_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `ended_at` TIMESTAMP NULL,
    `status` VARCHAR(10) NOT NULL DEFAULT 'active'
);

-- Add game_id to hand table
ALTER TABLE hand ADD COLUMN `game_id` VARCHAR(36) NOT NULL;
ALTER TABLE hand ADD CONSTRAINT `fk_hand_game` FOREIGN KEY (`game_id`) REFERENCES game (`id`);

-- Add game_id to card table
ALTER TABLE card ADD COLUMN `game_id` VARCHAR(36) NOT NULL;
ALTER TABLE card ADD CONSTRAINT `fk_card_game` FOREIGN KEY (`game_id`) REFERENCES game (`id`);

-- Create hand_history table for storing completed game hands
CREATE TABLE hand_history (
    `id` INT AUTO_INCREMENT PRIMARY KEY,
    `game_id` VARCHAR(36) NOT NULL,
    `player_id` INT NOT NULL,
    `equity` FLOAT,
    `is_muck` BOOLEAN NOT NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_game_id (`game_id`),
    INDEX idx_player_id (`player_id`)
);
