CREATE TABLE Users (
    id TEXT NOT NULL,  -- Based on the user's cookie
    display_name TEXT NOT NULL,  -- Arbitary, user specified
    PRIMARY KEY (user_id)
);

CREATE TABLE Games (
    id TEXT NOT NULL,  -- "Pronounceable", random combo of words
    status TEXT NOT NULL,  -- Enum: PENDING, PLAYING, FINISHED
    state BLOB NOT NULL,
    PRIMARY KEY (game_id)
);

CREATE TABLE GamePlayers (
    game_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL,  -- Enum: SPYMASTER, OPERATIVE
    team TEXT NOT NULL,  -- Enum: RED, BLUE
    FOREIGN KEY (game_id) REFERENCES Games(game_id),
    FOREIGN KEY (user_id) REFERENCES Users(user_id),
    PRIMARY KEY (game_id, user_id)
);

CREATE TABLE GameHistory (
    game_id TEXT NOT NULL,
    event_timestamp DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
    event BLOB NOT NULL,
    FOREIGN KEY (game_id) REFERENCES Games(game_id),
    PRIMARY KEY (game_id, event_timestamp)
);
