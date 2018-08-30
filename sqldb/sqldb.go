package sqldb

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"errors"
	"math/rand"
	"os"

	codenames "github.com/bcspragu/Codenames"

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
	createGameStmt = `INSERT INTO Games (id, status, state) VALUES (?, ?, ?)`
	gameExistsStmt = `SELECT EXISTS(SELECT 1 FROM Games WHERE id = ?)`

	getPendingGamesStmt = `SELECT id FROM Games WHERE status = 'Pending'`

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

		_, err = tx.Exec(createGameStmt, string(id), codenames.Pending, gsb)
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

func (s *DB) NewUser(u *codenames.User) (codenames.UserID, error) {
	type result struct {
		id  codenames.UserID
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

		id := codenames.RandomUserID(s.r)
		if err != nil {
			resChan <- &result{err: err}
			return
		}

		_, err = tx.Exec(createUserStmt, string(id), u.Name)
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
		return codenames.UserID(""), res.err
	}
	return res.id, nil
}

func (s *DB) PendingGames() ([]codenames.GameID, error) {
	return nil, codenames.ErrOperationNotImplemented
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
