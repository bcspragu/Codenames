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
	ErrRobotNotFound           = errors.New("codenames: robot not found")
	ErrGameNotFound            = errors.New("codenames: game not found")
)

type PlayerType string

const (
	UnknownPlayerType = PlayerType("")
	PlayerTypeHuman   = PlayerType("HUMAN")
	PlayerTypeRobot   = PlayerType("ROBOT")
)

func ToPlayerType(typ string) (PlayerType, bool) {
	switch typ {
	case "HUMAN":
		return PlayerTypeHuman, true
	case "ROBOT":
		return PlayerTypeRobot, true
	default:
		return UnknownPlayerType, false
	}
}

type PlayerID struct {
	PlayerType PlayerType `json:"player_type"`
	ID         string     `json:"id"`
}

func (p PlayerID) AsUserID() (UserID, bool) {
	if p.PlayerType != PlayerTypeHuman {
		return "", false
	}
	return UserID(p.ID), true
}

func (p PlayerID) AsRobotID() (RobotID, bool) {
	if p.PlayerType != PlayerTypeRobot {
		return "", false
	}
	return RobotID(p.ID), true
}

func (p PlayerID) String() string {
	return string(p.PlayerType) + ":" + p.ID
}

func (p PlayerID) IsUser(uID UserID) bool {
	return p.PlayerType == PlayerTypeHuman && p.ID == string(uID)
}

func (p PlayerID) IsRobot(rID RobotID) bool {
	return p.PlayerType == PlayerTypeRobot && p.ID == string(rID)
}

type UserID string

func (u UserID) AsPlayerID() PlayerID {
	return PlayerID{PlayerType: PlayerTypeHuman, ID: string(u)}
}

type RobotID string

func (r RobotID) AsPlayerID() PlayerID {
	return PlayerID{PlayerType: PlayerTypeRobot, ID: string(r)}
}

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

type Player struct {
	ID   PlayerID `json:"player_id"`
	Name string   `json:"name"`
}

func (p *Player) Clone() *Player {
	if p == nil {
		return nil
	}

	return &Player{
		ID:   p.ID,
		Name: p.Name,
	}
}

type Robot struct {
	ID   RobotID `json:"id"`
	Name string  `json:"name"`
}

func (r *Robot) Clone() *Robot {
	if r == nil {
		return nil
	}

	return &Robot{
		ID:   r.ID,
		Name: r.Name,
	}
}

type User struct {
	ID UserID `json:"id"`
	// Name is the name that gets displayed. It should arguably be called
	// DisplayName, but who's got time to type out all those letters.
	Name string `json:"name"`
}

func (u *User) Clone() *User {
	if u == nil {
		return nil
	}

	return &User{
		ID:   u.ID,
		Name: u.Name,
	}
}

type Game struct {
	ID        GameID     `json:"id"`
	CreatedBy UserID     `json:"created_by"`
	Status    GameStatus `json:"status"`
	State     *GameState `json:"state"`
}

func (g *Game) Clone() *Game {
	if g == nil {
		return nil
	}

	return &Game{
		ID:        g.ID,
		CreatedBy: g.CreatedBy,
		Status:    g.Status,
		State:     g.State.Clone(),
	}
}

type GameState struct {
	ActiveTeam     Team   `json:"active_team"`
	ActiveRole     Role   `json:"active_role"`
	Board          *Board `json:"board"`
	NumGuessesLeft int    `json:"num_guesses_left"`
	StartingTeam   Team   `json:"starting_team"`
}

func (gs *GameState) Clone() *GameState {
	if gs == nil {
		return nil
	}

	return &GameState{
		ActiveTeam:     gs.ActiveTeam,
		ActiveRole:     gs.ActiveRole,
		Board:          gs.Board.Clone(),
		NumGuessesLeft: gs.NumGuessesLeft,
		StartingTeam:   gs.StartingTeam,
	}
}

type PlayerRole struct {
	PlayerID     PlayerID `json:"player_id"`
	Team         Team     `json:"team"`
	Role         Role     `json:"role"`
	RoleAssigned bool     `json:"role_assigned"`
}

func (pr *PlayerRole) Clone() *PlayerRole {
	if pr == nil {
		return nil
	}

	return &PlayerRole{
		PlayerID:     pr.PlayerID,
		Team:         pr.Team,
		Role:         pr.Role,
		RoleAssigned: pr.RoleAssigned,
	}
}

func AllRolesFilled(prs []*PlayerRole) error {
	roleCount := make(map[Team]map[Role]int)
	for _, pr := range prs {
		if !pr.RoleAssigned {
			continue
		}

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
	NewUser(name string) (UserID, error)
	User(UserID) (*User, error)
	NewRobot(name string) (RobotID, error)
	Robot(RobotID) (*Robot, error)

	NewGame(*Game) (GameID, error)
	StartGame(gID GameID) error
	PendingGames() ([]GameID, error)
	Game(GameID) (*Game, error)
	JoinGame(GameID, PlayerID) error
	AssignRole(GameID, *PlayerRole) error

	PlayersInGame(gID GameID) ([]*PlayerRole, error)
	UpdateState(GameID, *GameState) error
	BatchPlayerNames([]PlayerID) (map[PlayerID]string, error)
	Player(id PlayerID) (string, error)
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
	return UserID("human_" + string(b))
}

func RandomRobotID(r *rand.Rand) RobotID {
	b := make([]byte, 64)
	for i := range b {
		b[i] = letters[r.Intn(len(letters))]
	}
	return RobotID("robot_" + string(b))
}

func randomWord(r *rand.Rand) string {
	var buf strings.Builder
	for _, word := range strings.Split(Words[r.Intn(len(Words))], "_") {
		buf.WriteString(strings.Title(word))
	}
	return buf.String()
}
