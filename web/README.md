# Codenames API Server

This directory holds the Codenames API server, which backs the ReactJS
frontend.

## API

The Codenames API Server serves an API over the usual HTTP/JSON spec, in a
generally RESTful way. The endpoints are as follows:

* `POST /api/user` - Creates a new user entity, required for playing games and
  generally doing anything.

  ```
  == Example Request ==
  POST /api/user
  {"name": "Testy McTesterson"}

  == Example Response ==
  Set-Cookie Authorization $SOME_ENCRYPTED_AUTH_TOKEN
  {"success": true}
  ```

  The important thing is to make sure the client is actually respecting the
  `Set-Cookie` response header, or auth won't actually work.

* `GET /api/user` - Loads information about the currently logged in user,
  returns `null` if there's no authentication header, or the account isn't
  found, etc, etc.

  ```
  == Example Request ==
  GET /api/user
  {} // Or no body at all is probably fine, just make sure the "Authorization"
  header is set.

  == Example Response (no user) ==
  null

  == Example Response (user is logged in) ==
  {"id": "abc123", "name": "Testy McTesterson"}
  ```

* `POST /api/game` - Creates a new pending game, and returns the ID of the
  newly created game.

  ```
  == Example Request ==
  POST /api/game
  {} // Or no body is likely fine.

  == Example Response ==
  {"id": "game123"}
  ```

* `GET /api/games` - Returns a list of all the games that haven't been started
  yet, basically a discount lobby.

  ```
  == Example Request ==
  GET /api/games
  {} // Again, no body is probably fine.

  == Example Response ==
  ["game123", "game456"] // A list of game IDs
  ```

* `GET /api/game/{id}` - Returns all the information we have about the game
  with the given ID.

  ```
  == Example Request ==
  GET /api/game/TheGameID123
  {} // No body or anything

  == Example Response ==
  {
    "id": "TheGameID123",
    "created_by": "user_id_123",
    "status": "PENDING",
    "state": {
      "active_team": "RED",
      "active_role": "SPYMASTER",
      "board": {
        "cards": [
          {"codeword": "watch", "agent": "UNKNOWN_AGENT", "revealed": false, "revealed_by": "NO_TEAM"},
          {"codeword": "time", "agent": "RED", "revealed": true, "revealed_by": "RED"},
          [ ... ]
        ]
      }
    }
  }
  ```
  Note that this endpoint requires an authenticated user, but doesn't require
  you to be in the game. If you aren't in the game (or are, but aren't the
  spymaster).

* `POST /api/game/{id}/join` - Joins the game with the given ID.
  ```
  == Example Request ==
  POST /api/game/TheGameID123/join
  {"team": "RED", "role": "SPYMASTER"}

  == Example Response ==
  {"success": true}
  ```
  There are a bunch of error conditions, like if the RED team already has a
  SPYMASTER (in the above example), or if the user doesn't have auth, or if the
  game has already started, etc. These will return an error message (no JSON
  for now) and a non-200 status code.

* `POST /api/game/{id}/start` - Kicks off the game with the given ID, can only
  be called by the person who created the game, once all roles have been
  filled.
  ```
  == Example Request ==
  POST /api/game/TheGameID123/start
  {"success": true}
  ```
  This will send down a WebSocket message to all connected players indicating
  that the game is on.

* `POST /api/game/{id}/clue` - Issues a clue, to be called by the active
  spymaster.
  ```
  == Example Request ==
  POST /api/game/TheGameID123/clue
  {"word": "muffins", "count": 3}

  == Example Response ==
  {"success": true}
  ```
  This will also send down a WebSocket message so that everyone gets the clue.


* `POST /api/game/{id}/guess` - Issues a (tentative or confirmed) guess from an
  operative player on the active team.
  ```
  == Example Request ==
  POST /api/game/TheGameID123/guess
  {"guess": "airplane", "confirmed": true}

  == Example Response ==
  {"success": true}
  ```
  The `"guess"` is the player's guess, from the cards still available on the
  board. The `"confirmed"` indicates whether or not they're actually putting in
  this vote, or just thinking about it. A guess will be selected once a
  majority of operatives on the team confirm guesses. Non-confirmed guesses are
  mostly so the UI can show what people are thinking.

## WebSockets

All of the live updates (game start, clues, votes, guesses, game over) are sent
via WebSockets, so that clients can get real-time pushes with the latest status
of the game. This is done by connecting to the WebSocket handler at
`/api/game/{id}/ws` once the player has joined the game.

All of the messages sent over WebSockets are JSON-formatted, and take the form:
```
{
  "action": "GAME_START | CLUE_GIVEN | PLAYER_VOTE | GUESS_GIVEN | GAME_END",
  ... other fields based on action ...
}
```

Look at the code in [web/msgs.go](/web/msgs.go) for details on fields and
message structure, or look at the handy guide below:

* `GAME_START`
  ```
  {
    "action": "GAME_START",
    "players": [
      {
        "player_id": "abc123",
        "name": "Test McTesterson",
        "team": "RED",
        "role": "SPYMASTER",
      },
      ... more users ...
    ],
    "game": {
      "id": "TheGameID123",
      "created_by": "user_id_123",
      "status": "PENDING",
      "state": {
        "active_team": "RED",
        "active_role": "SPYMASTER",
        "board": {
          "cards": [
            {"codeword": "watch", "agent": "UNKNOWN_AGENT", "revealed": false, "revealed_by": "NO_TEAM"},
            {"codeword": "time", "agent": "RED", "revealed": true, "revealed_by": "RED"},
            [ ... ]
          ]
        }
      }
    }
  }
  ```
* `CLUE_GIVEN`
  ```
  {
    "action": "CLUE_GIVEN",
    "clue": {
      "word": "helicopters",
      "count": 3
    },
    "team": "BLUE",
    "game": {... see above ...}
  }
  ```
* `PLAYER_VOTE`
  ```
  {
    "action": "PLAYER_VOTE",
    "user_id": "abc123",
    "guess": "blade",
    "confirmed": true
  }
  ```
* `GUESS_GIVEN`
  ```
  {
    "action": "GUESS_GIVEN",
    "guess": "blade",
    "team": "BLUE",
    "can_keep_guessing": true,
    "card": {
      "codeword": "blade",
      "agent": "BLUE",
      "revealed": true,
      "revealed_by": "BLUE"
    },
    "game": {... see above ...}
  }
  ```
* `GAME_END`
  ```
  {
    "action": "GAME_END",
    "winning_team": "BLUE",
    "game": {... see above ...}
  }
  ```

## Error Handling

If the response code is _not_ a `200 OK`, the response body will contain the
error message, which probably isn't human-readable, or at least not useful for
end clients.
