CREATE TABLE card (
    id INTEGER PRIMARY KEY,
    suit TEXT NOT NULL,
    rank TEXT NOT NULL,
    is_board BOOLEAN NOT NULL,
    UNIQUE(suit, rank)
);

CREATE TABLE player (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    serial TEXT NOT NULL
);

CREATE TABLE hand (
    id INTEGER PRIMARY KEY,
    player_id INTEGER NOT NULL,
    card_a_id INTEGER NOT NULL,
    card_b_id INTEGER NOT NULL,
    equity NUMERIC,
    is_muck BOOLEAN NOT NULL
);