CREATE TABLE players (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE matches (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    gamemode_id INTEGER NOT NULL REFERENCES gamemodes(id),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE gamemodes (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT
);

CREATE TABLE match_players (
    match_id INTEGER NOT NULL,
    player_id INTEGER NOT NULL,
    PRIMARY KEY (match_id, player_id)
);
