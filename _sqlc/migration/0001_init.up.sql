CREATE TABLE player (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE antenna (
    id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    serial TEXT UNIQUE NOT NULL,
    antenna_type_id INTEGER NOT NULL,
    player_id INTEGER,
    FOREIGN KEY (antenna_type_id) REFERENCES antenna_type (id)
);

CREATE TABLE antenna_type (
    id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
    name VARCHAR(10) NOT NULL
);

INSERT INTO antenna_type (name) VALUES ('player'), ('board'), ('muck'), ('unknown');

CREATE TABLE card (
    id INTEGER PRIMARY KEY,
    suit TEXT NOT NULL,
    rank TEXT NOT NULL,
    is_board BOOLEAN NOT NULL,
    hand_id INTEGER,
    serial TEXT NOT NULL,
    FOREIGN KEY (hand_id) REFERENCES hand (id),
    FOREIGN KEY (serial) REFERENCES antenna (serial),
    UNIQUE(suit, rank)
);

CREATE TABLE hand (
    id INTEGER PRIMARY KEY,
    player_id INTEGER NOT NULL,
    equity NUMERIC,
    is_muck BOOLEAN NOT NULL
);