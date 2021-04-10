package codenames

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"strings"
)

var (
	ErrOperationNotImplemented = errors.New("codenames: operation not implemented")
	ErrUserNotFound            = errors.New("codenames: user not found")
	ErrGameNotFound            = errors.New("codenames: game not found")
)

type PlayerType string

const (
	PlayerTypeHuman = PlayerType("human")
	PlayerTypeRobot = PlayerType("robot")
)

type PlayerID struct {
	PlayerType PlayerType `json:"player_type"`
	ID         string     `json:"id"`
}

func (p PlayerID) String() string {
	return string(p.PlayerType) + ":" + p.ID
}

func (p PlayerID) IsUser(uID UserID) bool {
	return p.PlayerType == PlayerTypeHuman && p.ID == string(uID)
}

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
	Finished = GameStatus("FINISHED")
)

type Role string

const (
	// NoRole is an error case.
	NoRole        = Role("")
	SpymasterRole = Role("SPYMASTER")
	OperativeRole = Role("OPERATIVE")
)

func ToRole(role string) (Role, bool) {
	switch role {
	case "SPYMASTER":
		return SpymasterRole, true
	case "OPERATIVE":
		return OperativeRole, true
	default:
		return NoRole, false
	}
}

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
	ActiveTeam     Team   `json:"active_team"`
	ActiveRole     Role   `json:"active_role"`
	Board          *Board `json:"board"`
	NumGuessesLeft int    `json:"num_guesses_left"`
	StartingTeam   Team   `json:"starting_team"`
}

type PlayerRole struct {
	PlayerID PlayerID `json:"player_id"`
	Team     Team     `json:"team"`
	Role     Role     `json:"role"`
}

func AllRolesFilled(prs []*PlayerRole) error {
	roleCount := make(map[Team]map[Role]int)
	for _, pr := range prs {
		rc, ok := roleCount[pr.Team]
		if !ok {
			rc = make(map[Role]int)
		}
		rc[pr.Role]++
		roleCount[pr.Team] = rc
	}
	count := func(team Team, role Role) int {
		rm, ok := roleCount[team]
		if !ok {
			return 0
		}
		return rm[role]
	}
	teams := []Team{BlueTeam, RedTeam}

	for _, t := range teams {
		switch n := count(t, SpymasterRole); n {
		case 0:
			return fmt.Errorf("team %q had no spymaster", t)
		case 1:
			// Good
		default:
			return fmt.Errorf("team %q somehow has %d spymasters", t, n)
		}
		if count(t, OperativeRole) == 0 {
			return fmt.Errorf("team %q had no operatives", t)
		}
	}
	return nil
}

type DB interface {
	NewUser(*User) (UserID, error)
	User(UserID) (*User, error)

	NewGame(*Game) (GameID, error)
	StartGame(gID GameID) error
	PendingGames() ([]GameID, error)
	Game(GameID) (*Game, error)
	JoinGame(GameID, *PlayerRole) error

	PlayersInGame(gID GameID) ([]*PlayerRole, error)
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

func RandomPlayerID(r *rand.Rand) string {
	b := make([]byte, 64)
	for i := range b {
		b[i] = letters[r.Intn(len(letters))]
	}
	return string(b)
}

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
