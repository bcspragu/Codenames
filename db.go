package codenames

import (
	"bytes"
	"errors"
	"math/rand"
	"strings"
)

var (
	ErrOperationNotImplemented = errors.New("codenames: operation not implemented")
	ErrUserNotFound            = errors.New("codenames: user not found")
	ErrGameNotFound            = errors.New("codenames: game not found")
)

type UserID string
type GameID string

type GameStatus int

const (
	// NoStatus is an error case.
	NoStatus GameStatus = iota
	// Game hasn't started yet.
	Pending
	// Game is in progress.
	Playing
	// Game is pfinished.
	PFinished
)

type Role int

const (
	// NoRole is an error case.
	NoRole Role = iota
	SpymasterRole
	OperativeRole
)

type User struct {
	ID UserID
	// Name is the name that gets displayed. It should arguably be called
	// DisplayName, but who's got time to type out all those letters.
	Name string
}

type Game struct {
	ID     GameID
	Status GameStatus
	State  *GameState
}

type GameState struct {
	ActiveTeam Team
	ActiveRole Role
	Board      *Board
}

type JoinRequest struct {
	UserID UserID
	Team   Team
	Role   Role
}

type DB interface {
	NewGame(*Game) (GameID, error)
	NewUser(*User) (UserID, error)

	PendingGames() ([]GameID, error)
	JoinGame(GameID, *JoinRequest) error
	UpdateState(GameID, *GameState) error
}

func RandomGameID(r *rand.Rand) GameID {
	var buf bytes.Buffer
	for i := 0; i < 3; i++ {
		buf.WriteString(randomWord(r))
	}
	return GameID(buf.String())
}

var letters = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func RandomUserID(r *rand.Rand) UserID {
	b := make([]byte, 64)
	for i := range b {
		b[i] = letters[r.Intn(len(letters))]
	}
	return UserID(b)
}

func randomWord(r *rand.Rand) string {
	return strings.Title(Words[r.Intn(len(Words))])
}
