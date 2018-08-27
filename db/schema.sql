CREATE TABLE Users (
    user_id TEXT,  -- Based on the user's cookie
    display_name TEXT,  -- Arbitary, user specified
    PRIMARY KEY (user_id)
);

CREATE TABLE Games (
    game_id TEXT,  -- Pronounceable, random combo of words
    status TEXT,  -- Enum: PENDING, PLAYING, FINISHED
    state BLOB,
    PRIMARY KEY (game_id)
);

CREATE TABLE GamePlayers (
    game_id TEXT,
    user_id TEXT,
    role TEXT,  -- Enum: SPYMASTER, OPERATIVE
    team TEXT,  -- Enum: RED, BLUE
    FOREIGN KEY (game_id) REFERENCES Games(game_id),
    FOREIGN KEY (user_id) REFERENCES Users(user_id),
    PRIMARY KEY (game_id, user_id)
);

CREATE TABLE GameHistory (
    game_id TEXT,
    event_timestamp DATETIME,
    event BLOB,
    FOREIGN KEY (game_id) REFERENCES Games(game_id),
    PRIMARY KEY (game_id, event_timestamp)
);
