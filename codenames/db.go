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

type GameStatus string

const (
	// NoStatus is an error case.
	NoStatus = GameStatus("")
	// Game hasn't started yet.
	Pending = GameStatus("PENDING")
	// Game is in progress.
	Playing = GameStatus("PLAYING")
	// Game is pfinished.
	PFinished = GameStatus("PFINISHED")
)

type Role string

const (
	// NoRole is an error case.
	NoRole        = Role("")
	SpymasterRole = Role("SPYMASTER")
	OperativeRole = Role("OPERATIVE")
)

type User struct {
	ID UserID `json:"id"`
	// Name is the name that gets displayed. It should arguably be called
	// DisplayName, but who's got time to type out all those letters.
	Name string `json:"name"`
}

type Game struct {
	ID        GameID     `json:"id"`
	CreatedBy UserID     `json:"created_by"`
	Status    GameStatus `json:"status"`
	State     *GameState `json:"state"`
}

type GameState struct {
	ActiveTeam Team   `json:"active_team"`
	ActiveRole Role   `json:"active_role"`
	Board      *Board `json:"board"`
}

type JoinRequest struct {
	UserID UserID
	Team   Team
	Role   Role
}

type DB interface {
	NewGame(*Game) (GameID, error)
	NewUser(*User) (UserID, error)

	User(UserID) (*User, error)

	PendingGames() ([]GameID, error)
	Game(GameID) (*Game, error)
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
