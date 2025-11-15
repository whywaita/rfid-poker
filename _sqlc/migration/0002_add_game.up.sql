-- Add game table for tracking individual games with UUID
CREATE TABLE game (
    `id` VARCHAR(36) PRIMARY KEY,
    `started_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `ended_at` TIMESTAMP NULL,
    `status` VARCHAR(10) NOT NULL DEFAULT 'active'
);

-- Add game_id to hand table
ALTER TABLE hand ADD COLUMN `game_id` VARCHAR(36);
ALTER TABLE hand ADD FOREIGN KEY (`game_id`) REFERENCES game (`id`);

-- Add game_id to card table
ALTER TABLE card ADD COLUMN `game_id` VARCHAR(36);
ALTER TABLE card ADD FOREIGN KEY (`game_id`) REFERENCES game (`id`);
