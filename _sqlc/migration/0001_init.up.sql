CREATE TABLE player (
    `id` INT AUTO_INCREMENT PRIMARY KEY,
    `name` VARCHAR(255) NOT NULL
);

CREATE TABLE antenna_type (
    `id` INT AUTO_INCREMENT PRIMARY KEY,
    `name` VARCHAR(10) NOT NULL UNIQUE
);

INSERT INTO antenna_type (`name`) VALUES ('player'), ('board'), ('muck'), ('unknown');

CREATE TABLE antenna (
    `id` INT AUTO_INCREMENT PRIMARY KEY,
    `serial` VARCHAR(255) UNIQUE NOT NULL,
    `antenna_type_id` INT NOT NULL,
    `player_id` INT,
    UNIQUE (`serial`),
    FOREIGN KEY (`antenna_type_id`) REFERENCES antenna_type (id)
);

CREATE TABLE hand (
    `id` INT AUTO_INCREMENT PRIMARY KEY,
    `player_id` INT NOT NULL,
    `equity` FLOAT,
    `is_muck` BOOLEAN NOT NULL
);

CREATE TABLE card (
    `id` INT AUTO_INCREMENT PRIMARY KEY,
    `card_suit` VARCHAR(255) NOT NULL,
    `card_rank` VARCHAR(255) NOT NULL,
    `is_board` BOOLEAN NOT NULL,
    `hand_id` INT,
    `serial` VARCHAR(255) NOT NULL,
    FOREIGN KEY (`serial`) REFERENCES antenna (`serial`),
    UNIQUE(`card_suit`, `card_rank`)
);