package sqldb

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"

	"github.com/bcspragu/Codenames/codenames"

	_ "github.com/mattn/go-sqlite3"
)

var ()

var (
	// Game statements
	createGameStmt      = `INSERT INTO Games (id, status, creator_id, state) VALUES (?, ?, ?, ?)`
	gameExistsStmt      = `SELECT EXISTS(SELECT 1 FROM Games WHERE id = ?)`
	getGameStmt         = `SELECT id, status, creator_id, state FROM Games WHERE id = ?`
	getPendingGamesStmt = `SELECT id FROM Games WHERE status = 'PENDING' ORDER BY id`
	startGameStmt       = `
UPDATE Games
SET status = 'PLAYING'
WHERE id = ?`
	updateGameStateStmt = `
UPDATE Games
SET state = ?
WHERE id = ?`

	// User statements
	createUserStmt = `INSERT INTO Users (id, display_name) VALUES (?, ?)`
	getUserStmt    = `SELECT id, display_name FROM Users WHERE id = ?`

	// Player (e.g. user or AI) statements
	getUserPlayerStmt = `SELECT id FROM Players WHERE user_id = ?`
	getAIPlayerStmt   = `SELECT id FROM Players WHERE ai_id = ?`
	createPlayerStmt  = `INSERT INTO Players (id, user_id, ai_id) VALUES (?, ?, ?)`

	// Game player (e.g. Game <-> Player join table) statements
	joinGameStmt   = `INSERT INTO GamePlayers (game_id, player_id, role, team) VALUES (?, ?, ?, ?)`
	getGamePlayers = `
SELECT Players.user_id, Players.ai_id, GamePlayers.role, GamePlayers.team
FROM GamePlayers
JOIN Players
	ON GamePlayers.player_id = Players.id
WHERE GamePlayers.game_id = ?`

	// Game history statements (currently unused)
	updateGameHistoryStmt = `INSERT INTO GameHistory (game_id, event) VALUES (?, ?)`
)

// DB implements the Codenames database API, backed by a SQLite database.
// NOTE: Since the database doesn't support concurrent writers, we don't
// actually hold the *sql.DB in this struct, we force all callers to get a
// handle via channels.
type DB struct {
	dbChan   chan func(*sql.DB)
	doneChan chan struct{}
	closeFn  func() error
	r        *rand.Rand
}

// New creates a new *DB that is stored on disk at the given filename.
func New(fn string, r *rand.Rand) (*DB, error) {
	if _, err := os.Stat(fn); os.IsNotExist(err) {
		return nil, errors.New("DB needs to be initialized")
	}
	sdb, err := sql.Open("sqlite3", fn)
	if err != nil {
		return nil, err
	}

	db := &DB{
		dbChan:   make(chan func(*sql.DB)),
		doneChan: make(chan struct{}),
		closeFn: func() error {
			return sdb.Close()
		},
		r: r,
	}
	go db.run(sdb)
	return db, nil
}

// run handles all database calls, and ensures that only one thing is happening
// against the database at a time.
func (s *DB) run(sdb *sql.DB) {
	for {
		select {
		case dbFn := <-s.dbChan:
			dbFn(sdb)
		case <-s.doneChan:
			sdb.Close()
			return
		}
	}
}

func (s *DB) Close() error {
	close(s.doneChan)
	return s.closeFn()
}

func (s *DB) NewGame(g *codenames.Game) (codenames.GameID, error) {
	gsb, err := gameStateBytes(g.State)
	if err != nil {
		return "", fmt.Errorf("failed to serialize game state: %w", err)
	}

	type result struct {
		id  codenames.GameID
		err error
	}

	resChan := make(chan *result)
	s.dbChan <- func(sdb *sql.DB) {
		tx, err := sdb.Begin()
		if err != nil {
			resChan <- &result{err: err}
			return
		}
		defer tx.Rollback()

		id, err := s.uniqueID(tx)
		if err != nil {
			resChan <- &result{err: err}
			return
		}

		_, err = tx.Exec(createGameStmt, string(id), codenames.Pending, string(g.CreatedBy), gsb)
		if err != nil {
			resChan <- &result{err: err}
			return
		}

		if err := tx.Commit(); err != nil {
			resChan <- &result{err: err}
			return
		}
		resChan <- &result{id: id}
	}

	res := <-resChan
	if res.err != nil {
		return codenames.GameID(""), res.err
	}
	return res.id, nil
}

func (s *DB) Game(gID codenames.GameID) (*codenames.Game, error) {
	type result struct {
		game *codenames.Game
		err  error
	}

	resChan := make(chan *result)
	s.dbChan <- func(sdb *sql.DB) {
		tx, err := sdb.Begin()
		if err != nil {
			resChan <- &result{err: err}
			return
		}
		defer tx.Rollback()

		var (
			g   codenames.Game
			gsb []byte
		)
		if err := tx.QueryRow(getGameStmt, string(gID)).Scan(&g.ID, &g.Status, &g.CreatedBy, &gsb); err != nil {
			resChan <- &result{err: err}
			return
		}

		if err := tx.Commit(); err != nil {
			resChan <- &result{err: err}
			return
		}

		if g.State, err = gameStateFromBytes(gsb); err != nil {
			resChan <- &result{err: err}
			return
		}
		resChan <- &result{game: &g}
	}

	res := <-resChan
	if res.err != nil {
		return nil, res.err
	}
	return res.game, nil
}

func (s *DB) NewUser(u *codenames.User) (codenames.UserID, error) {
	type result struct {
		id  codenames.UserID
		err error
	}

	resChan := make(chan *result)
	s.dbChan <- func(sdb *sql.DB) {
		id := codenames.RandomUserID(s.r)
		_, err := sdb.Exec(createUserStmt, string(id), u.Name)
		if err != nil {
			resChan <- &result{err: err}
			return
		}

		resChan <- &result{id: id}
	}

	res := <-resChan
	if res.err != nil {
		return codenames.UserID(""), res.err
	}
	return res.id, nil
}

func (s *DB) User(id codenames.UserID) (*codenames.User, error) {
	type result struct {
		user *codenames.User
		err  error
	}

	resChan := make(chan *result)
	s.dbChan <- func(sdb *sql.DB) {
		var u codenames.User
		err := sdb.QueryRow(getUserStmt, string(id)).Scan(&u.ID, &u.Name)
		if err == sql.ErrNoRows {
			resChan <- &result{err: codenames.ErrUserNotFound}
			return
		} else if err != nil {
			resChan <- &result{err: err}
			return
		}

		resChan <- &result{user: &u}
	}

	res := <-resChan
	if res.err != nil {
		return nil, res.err
	}
	return res.user, nil
}

func (s *DB) PendingGames() ([]codenames.GameID, error) {
	type result struct {
		ids []codenames.GameID
		err error
	}

	resChan := make(chan *result)
	s.dbChan <- func(sdb *sql.DB) {
		rows, err := sdb.Query(getPendingGamesStmt)
		if err != nil {
			resChan <- &result{err: err}
			return
		}
		defer rows.Close()

		var ids []codenames.GameID
		for rows.Next() {
			var id codenames.GameID
			if err := rows.Scan(&id); err != nil {
				resChan <- &result{err: err}
				return
			}
			ids = append(ids, id)
		}

		if err := rows.Err(); err != nil {
			resChan <- &result{err: err}
			return
		}

		resChan <- &result{ids: ids}
	}
	res := <-resChan
	if res.err != nil {
		return nil, res.err
	}

	return res.ids, nil
}

func (s *DB) PlayersInGame(gID codenames.GameID) ([]*codenames.PlayerRole, error) {
	type result struct {
		prs []*codenames.PlayerRole
		err error
	}
	resChan := make(chan *result)

	s.dbChan <- func(sdb *sql.DB) {
		rows, err := sdb.Query(getGamePlayers, gID)
		if err != nil {
			resChan <- &result{err: fmt.Errorf("failed to query for game players: %w", err)}
			return
		}
		defer rows.Close()

		var prs []*codenames.PlayerRole
		for rows.Next() {
			var (
				pr     codenames.PlayerRole
				userID sql.NullString
				aiID   sql.NullString
			)
			if err := rows.Scan(&userID, &aiID, &pr.Role, &pr.Team); err != nil {
				resChan <- &result{err: fmt.Errorf("failed to scan game player: %w", err)}
				return
			}
			if userID.Valid && aiID.Valid {
				resChan <- &result{err: fmt.Errorf("both user_id and ai_id were set: %q, %q", userID.String, aiID.String)}
				return
			}
			if !userID.Valid && !aiID.Valid {
				resChan <- &result{err: errors.New("neither of user_id or ai_id were set")}
				return
			}
			if userID.Valid {
				pr.PlayerID = codenames.PlayerID{
					PlayerType: codenames.PlayerTypeHuman,
					ID:         userID.String,
				}
			}
			if aiID.Valid {
				pr.PlayerID = codenames.PlayerID{
					PlayerType: codenames.PlayerTypeRobot,
					ID:         aiID.String,
				}
			}
			prs = append(prs, &pr)
		}

		if err := rows.Err(); err != nil {
			resChan <- &result{err: fmt.Errorf("error scanning rows: %w", err)}
			return
		}

		resChan <- &result{prs: prs}
		return
	}
	res := <-resChan
	if res.err != nil {
		return nil, res.err
	}
	return res.prs, nil
}

func (s *DB) JoinGame(gID codenames.GameID, req *codenames.PlayerRole) error {
	// First, see if a player entity already exists for this player.
	pID, err := s.player(req.PlayerID)
	if err == sql.ErrNoRows {
		if pID, err = s.createPlayer(req.PlayerID); err != nil {
			return fmt.Errorf("failed to create player: %w", err)
		}
	}
	if err != nil {
		return fmt.Errorf("failed to load player: %w", err)
	}

	// If we're here, we've got a player ID and we can add them to the game.
	resChan := make(chan error)
	s.dbChan <- func(sdb *sql.DB) {
		_, err := sdb.Exec(joinGameStmt, gID, pID, req.Role, req.Team)
		resChan <- err
	}

	if err := <-resChan; err != nil {
		return fmt.Errorf("failed to join game: %w", err)
	}
	return nil
}

func (s *DB) createPlayer(id codenames.PlayerID) (string, error) {
	type result struct {
		id  string
		err error
	}

	resChan := make(chan *result)
	s.dbChan <- func(sdb *sql.DB) {
		pID := codenames.RandomPlayerID(s.r)
		var userID, aiID sql.NullString
		switch id.PlayerType {
		case codenames.PlayerTypeHuman:
			userID.Valid = true
			userID.String = id.ID
		case codenames.PlayerTypeRobot:
			aiID.Valid = true
			aiID.String = id.ID
		default:
			resChan <- &result{err: fmt.Errorf("unknown player type %q", id.PlayerType)}
			return
		}
		if _, err := sdb.Exec(createPlayerStmt, pID, userID, aiID); err != nil {
			resChan <- &result{err: fmt.Errorf("failed to insert player", err)}
			return
		}
		resChan <- &result{id: pID}
	}

	res := <-resChan
	if res.err != nil {
		return "", res.err
	}
	return res.id, nil
}

func (s *DB) player(id codenames.PlayerID) (string, error) {
	type result struct {
		id  string
		err error
	}

	resChan := make(chan *result)
	s.dbChan <- func(sdb *sql.DB) {
		var stmt string
		switch id.PlayerType {
		case codenames.PlayerTypeHuman:
			stmt = getUserPlayerStmt
		case codenames.PlayerTypeRobot:
			stmt = getAIPlayerStmt
		default:
			resChan <- &result{err: fmt.Errorf("unknown player type %q", id.PlayerType)}
			return
		}
		var outID string
		if err := sdb.QueryRow(stmt, id.ID).Scan(&outID); err != nil {
			resChan <- &result{err: err}
			return
		}
		resChan <- &result{id: outID}
	}

	res := <-resChan
	if res.err != nil {
		return "", res.err
	}
	return res.id, nil
}

func (s *DB) BatchPlayerNames(pIDs []codenames.PlayerID) (map[codenames.PlayerID]string, error) {
	type result struct {
		names map[codenames.PlayerID]string
		err   error
	}

	var userIDArgs, aiIDArgs []interface{}
	for _, pID := range pIDs {
		switch pID.PlayerType {
		case codenames.PlayerTypeHuman:
			userIDArgs = append(userIDArgs, pID.ID)
		case codenames.PlayerTypeRobot:
			aiIDArgs = append(aiIDArgs, pID.ID)
		default:
			return nil, fmt.Errorf("unknown player type %q", pID.PlayerType)
		}
	}

	q := fmt.Sprintf(`
SELECT Users.display_name, Players.user_id, "user"
FROM Players
JOIN Users
  ON Users.id = Players.user_id
WHERE Users.id IN %s
UNION ALL
SELECT AIs.display_name, Players.ai_id, "ai"
FROM Players
JOIN AIs
  ON AIs.id = Players.ai_id
WHERE AIs.id IN %s`, groupedArgs(len(userIDArgs)), groupedArgs(len(aiIDArgs)))

	var allIDArgs []interface{}
	allIDArgs = append(userIDArgs, aiIDArgs...)

	resChan := make(chan *result)
	s.dbChan <- func(sdb *sql.DB) {
		rows, err := sdb.Query(q, allIDArgs...)
		if err != nil {
			resChan <- &result{err: fmt.Errorf("failed to query names: %w", err)}
			return
		}

		out := make(map[codenames.PlayerID]string)
		for rows.Next() {
			var name, id, typ string
			if err := rows.Scan(&name, &id, &typ); err != nil {
				resChan <- &result{err: fmt.Errorf("error scanning row: %w", err)}
				return
			}
			var playerType codenames.PlayerType
			switch typ {
			case "user":
				playerType = codenames.PlayerTypeHuman
			case "ai":
				playerType = codenames.PlayerTypeRobot
			default:
				resChan <- &result{err: fmt.Errorf("unexpected player type %q", typ)}
				return
			}
			pID := codenames.PlayerID{PlayerType: playerType, ID: id}
			out[pID] = name
		}

		if err := rows.Err(); err != nil {
			resChan <- &result{err: fmt.Errorf("error scanning rows: %w", err)}
			return
		}

		resChan <- &result{names: out}
	}

	res := <-resChan
	if res.err != nil {
		return nil, res.err
	}
	return res.names, nil
}

func groupedArgs(n int) string {
	if n <= 0 {
		return "(NULL)"
	}
	return "(?" + strings.Repeat(",?", n-1) + ")"
}

func (s *DB) StartGame(gID codenames.GameID) error {
	resChan := make(chan error)
	s.dbChan <- func(sdb *sql.DB) {
		_, err := sdb.Exec(startGameStmt, gID)
		resChan <- err
	}

	if err := <-resChan; err != nil {
		return fmt.Errorf("failed to mark game started: %w", err)
	}
	return nil
}

func (s *DB) UpdateState(gID codenames.GameID, gs *codenames.GameState) error {
	gsb, err := gameStateBytes(gs)
	if err != nil {
		return fmt.Errorf("failed to serialize game state: %w", err)
	}

	resChan := make(chan error)
	s.dbChan <- func(sdb *sql.DB) {
		_, err := sdb.Exec(updateGameStateStmt, gsb, gID)
		resChan <- err
	}

	if err := <-resChan; err != nil {
		return fmt.Errorf("failed to update game state: %w", err)
	}
	return nil
}

func (s *DB) uniqueID(tx *sql.Tx) (codenames.GameID, error) {
	i := 0
	var id codenames.GameID
	for {
		id = codenames.RandomGameID(s.r)
		var n int
		if err := tx.QueryRow(gameExistsStmt, id).Scan(&n); err != nil {
			return codenames.GameID(""), err
		}
		if n == 0 {
			break
		}
		i++
		if i >= 100 {
			return codenames.GameID(""), errors.New("tried 100 random IDs, all were taken, which seems fishy")
		}
	}
	return id, nil
}

func gameStateBytes(s *codenames.GameState) ([]byte, error) {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(&s)
	return buf.Bytes(), err
}

func gameStateFromBytes(dat []byte) (*codenames.GameState, error) {
	var gs codenames.GameState
	if err := gob.NewDecoder(bytes.NewReader(dat)).Decode(&gs); err != nil {
		return nil, fmt.Errorf("failed to load game state: %w", err)
	}
	return &gs, nil
}
