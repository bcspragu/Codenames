package sqldb

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"errors"
	"fmt"
	"math/rand"
	"os"

	"github.com/bcspragu/Codenames/codenames"

	_ "github.com/mattn/go-sqlite3"
)

var (
	dbToGameStatus = map[string]codenames.GameStatus{
		"Pending":   codenames.Pending,
		"Playing":   codenames.Playing,
		"PFinished": codenames.PFinished,
	}
	gameStatusToDB = map[codenames.GameStatus]string{
		codenames.Pending:   "Pending",
		codenames.Playing:   "Playing",
		codenames.PFinished: "PFinished",
	}
)

var (
	createUserStmt = `INSERT INTO Users (id, display_name) VALUES (?, ?)`
	createGameStmt = `INSERT INTO Games (id, status, creator_id, state) VALUES (?, ?, ?, ?)`
	gameExistsStmt = `SELECT EXISTS(SELECT 1 FROM Games WHERE id = ?)`
	getGameStmt    = `SELECT id, status, creator_id, state FROM Games WHERE id = ?`

	getUserStmt = `SELECT id, display_name FROM Users WHERE id = ?`

	getPendingGamesStmt = `SELECT id FROM Games WHERE status = 'PENDING' ORDER BY id`

	joinGameStmt        = `INSERT INTO GamePlayers (game_id, user_id, role, team) VALUES (?, ?, ?, ?)`
	updateGameStateStmt = `INSERT INTO GameHistory (game_id, event) VALUES (?, ?)`
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

		gsb, err := gameStateBytes(g.State)
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

func (s *DB) JoinGame(_ codenames.GameID, _ *codenames.JoinRequest) error {
	return codenames.ErrOperationNotImplemented
}

func (s *DB) UpdateState(_ codenames.GameID, _ *codenames.GameState) error {
	return codenames.ErrOperationNotImplemented
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
