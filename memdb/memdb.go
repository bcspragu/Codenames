package memdb

import (
	"fmt"

	"github.com/bcspragu/Codenames/codenames"
)

type idNamespace string

const (
	gameID  = idNamespace("game")
	userID  = idNamespace("user")
	robotID = idNamespace("robot")
)

type DB struct {
	ids         map[idNamespace]int
	games       map[codenames.GameID]*codenames.Game
	users       map[codenames.UserID]*codenames.User
	robots      map[codenames.RobotID]*codenames.Robot
	playerRoles map[codenames.GameID][]*codenames.PlayerRole
}

func New() *DB {
	return &DB{
		ids:         make(map[idNamespace]int),
		games:       make(map[codenames.GameID]*codenames.Game),
		users:       make(map[codenames.UserID]*codenames.User),
		robots:      make(map[codenames.RobotID]*codenames.Robot),
		playerRoles: make(map[codenames.GameID][]*codenames.PlayerRole),
	}
}

func (db *DB) NewGame(g *codenames.Game) (codenames.GameID, error) {
	gID := codenames.GameID(db.newID(gameID))

	gc := g.Clone()
	gc.ID = gID
	gc.Status = codenames.Pending
	db.games[gID] = gc
	db.playerRoles[gID] = []*codenames.PlayerRole{}

	return gID, nil
}

func (db *DB) Game(gID codenames.GameID) (*codenames.Game, error) {
	g, ok := db.games[gID]
	if !ok {
		return nil, codenames.ErrGameNotFound
	}

	return g.Clone(), nil
}

func (db *DB) NewUser(name string) (codenames.UserID, error) {
	uID := codenames.UserID(db.newID(userID))

	u := &codenames.User{ID: uID, Name: name}
	u.ID = uID
	db.users[uID] = u

	return uID, nil
}

func (db *DB) User(uID codenames.UserID) (*codenames.User, error) {
	u, ok := db.users[uID]
	if !ok {
		return nil, codenames.ErrUserNotFound
	}

	return u.Clone(), nil
}

func (db *DB) NewRobot(name string) (codenames.RobotID, error) {
	rID := codenames.RobotID(db.newID(robotID))

	r := &codenames.Robot{ID: rID, Name: name}
	r.ID = rID
	db.robots[rID] = r

	return rID, nil
}

func (db *DB) Robot(rID codenames.RobotID) (*codenames.Robot, error) {
	r, ok := db.robots[rID]
	if !ok {
		return nil, codenames.ErrRobotNotFound
	}

	return r.Clone(), nil
}

func (db *DB) PendingGames() ([]codenames.GameID, error) {
	var pending []codenames.GameID
	for _, g := range db.games {
		if g.Status == codenames.Pending {
			pending = append(pending, g.ID)
		}
	}
	return pending, nil
}

func (db *DB) PlayersInGame(gID codenames.GameID) ([]*codenames.PlayerRole, error) {
	prs, ok := db.playerRoles[gID]
	if !ok {
		return nil, codenames.ErrGameNotFound
	}

	return clonePRs(prs), nil
}

func (db *DB) Player(pID codenames.PlayerID) (string, error) {
	if pID.PlayerType != codenames.PlayerTypeHuman {
		return "", fmt.Errorf("player type %q not supported for memdb, only humans for now", pID.PlayerType)
	}

	for _, u := range db.users {
		if pID.IsUser(u.ID) {
			return "", nil
		}
	}

	return "", fmt.Errorf("player %+v not found", pID)
}

func clonePRs(prs []*codenames.PlayerRole) []*codenames.PlayerRole {
	out := make([]*codenames.PlayerRole, len(prs))
	for i, pr := range prs {
		out[i] = pr.Clone()
	}
	return out
}

func (db *DB) JoinGame(gID codenames.GameID, pID codenames.PlayerID) error {
	prs, ok := db.playerRoles[gID]
	if !ok {
		return codenames.ErrGameNotFound
	}

	// The SQLite implementation would fail if the player was
	// already in the game, so we should do the samee.
	for _, pr := range prs {
		if pr.PlayerID == pID {
			return fmt.Errorf("player %+v is already in game %q", pID, gID)
		}
	}

	// If we're here, we can add the player to the game.
	prs = append(prs, &codenames.PlayerRole{
		PlayerID: pID,
	})
	db.playerRoles[gID] = prs

	return nil
}

func (db *DB) AssignRole(gID codenames.GameID, req *codenames.PlayerRole) error {
	prs, ok := db.playerRoles[gID]
	if !ok {
		return codenames.ErrGameNotFound
	}

	for _, pr := range prs {
		if pr.PlayerID == req.PlayerID {
			pr.Role = req.Role
			pr.Team = req.Team
			pr.RoleAssigned = true
			return nil
		}
	}

	return fmt.Errorf("player %+v was not found in game %q", req.PlayerID, gID)
}

func (db *DB) BatchPlayerNames(pIDs []codenames.PlayerID) (map[codenames.PlayerID]string, error) {
	out := make(map[codenames.PlayerID]string)
	for _, pID := range pIDs {
		if pID.PlayerType != codenames.PlayerTypeHuman {
			return nil, fmt.Errorf("player type %q not supported for memdb, only humans for now", pID.PlayerType)
		}

		u, ok := db.users[codenames.UserID(pID.ID)]
		if !ok {
			return nil, fmt.Errorf("player %q was not found: %w", pID.ID, codenames.ErrUserNotFound)
		}

		out[pID] = u.Name
	}

	return out, nil
}

func (db *DB) StartGame(gID codenames.GameID) error {
	return db.updateGame(gID, func(g *codenames.Game) {
		g.Status = codenames.Playing
	})
}

func (db *DB) UpdateState(gID codenames.GameID, gs *codenames.GameState) error {
	return db.updateGame(gID, func(g *codenames.Game) {
		g.State = gs.Clone()
	})
}

func (db *DB) updateGame(gID codenames.GameID, update func(*codenames.Game)) error {
	g, ok := db.games[gID]
	if !ok {
		return codenames.ErrGameNotFound
	}
	update(g)
	return nil
}

func (db *DB) newID(ns idNamespace) string {
	idx := db.ids[ns]
	id := fmt.Sprintf("%s_%d", ns, idx)
	db.ids[ns]++
	return id
}
