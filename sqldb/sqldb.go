package sqldb

import (
	"database/sql"
	"errors"
	"math/rand"

	"github.com/bcspragu/Codenames/db"

	_ "github.com/mattn/go-sqlite3"
)

// DB implements the Codenames database API, backed by a SQLite database.
// NOTE: Since the database doesn't support concurrent writers, we don't
// actually hold the *sql.DB in this struct, we force all callers to get a
// handle via channels.
type DB struct {
	dbChan   chan func(*sql.DB)
	doneChan chan struct{}
	closeFn  func() error
	src      rand.Source
}

// New creates a new *DB that is stored on disk at the given filename.
func New(fn string, src rand.Source) (*DB, error) {
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
		src: src,
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

func (s *DB) NewGame(_ *db.Game) (db.GameID, error) {
	return db.GameID(""), errors.New("not implemented")
}

func (s *DB) NewUser(_ *db.User) (db.UserID, error) {
	return db.UserID(""), errors.New("not implemented")
}

func (s *DB) JoinGame(_ db.GameID, _ *db.JoinRequest) error {
	return errors.New("not implemented")
}

func (s *DB) UpdateState(_ db.GameID, _ *db.GameState) error {
	return errors.New("not implemented")
}
