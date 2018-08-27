package db

import (
	"errors"

	codenames "github.com/bcspragu/Codenames"
)

var (
	ErrOperationNotImplemented = errors.New("radiotation: operation not implemented")
	ErrUserNotFound            = errors.New("radiotation: user not found")
	ErrRoomNotFound            = errors.New("radiotation: room not found")
	ErrQueueNotFound           = errors.New("radiotation: queue not found")
	ErrNoTracksInQueue         = errors.New("radiotation: no tracks in queue")
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

type Team int

const (
	// NoTeam is an error case.
	NoTeam Team = iota
	RedTeam
	BlueTeam
)

type Role int

const (
	// NoRole is an error case.
	NoRole Role = iota
	Spymaster
	Operative
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
	State  GameState
}

type GameState struct {
	ActiveTeam Team
	ActiveRole Role
	Board      *codenames.Board
}

type JoinRequest struct {
	UserID UserID
	Team   Team
	Role   Role
}

type DB interface {
	NewGame(*Game) (GameID, error)
	NewUser(*User) (UserID, error)

	JoinGame(GameID, *JoinRequest) error
	UpdateState(GameID, *GameState) error
}
