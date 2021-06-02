package web

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"

	"github.com/bcspragu/Codenames/boardgen"
	"github.com/bcspragu/Codenames/codenames"
	"github.com/bcspragu/Codenames/consensus"
	"github.com/bcspragu/Codenames/game"
	"github.com/bcspragu/Codenames/httperr"
	"github.com/bcspragu/Codenames/hub"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/websocket"
)

const (
	maxOperativesPerTeam = 10
)

type Srv struct {
	sc        *securecookie.SecureCookie
	hub       *hub.Hub
	mux       *mux.Router
	db        codenames.DB
	r         *rand.Rand
	ws        *websocket.Upgrader
	consensus *consensus.Guesser
}

// New returns an initialized server.
func New(db codenames.DB, r *rand.Rand, sc *securecookie.SecureCookie) *Srv {
	s := &Srv{
		sc:        sc,
		hub:       hub.New(),
		db:        db,
		r:         r,
		ws:        &websocket.Upgrader{}, // use default options, for now
		consensus: consensus.New(),
	}

	s.mux = s.initMux()

	return s
}

type handlerFunc func(w http.ResponseWriter, r *http.Request) error

func (s *Srv) initMux() *mux.Router {
	m := mux.NewRouter()

	handlers := []struct {
		path        string
		method      string
		handlerFunc handlerFunc
	}{
		// New user.
		{
			path:        "/api/user",
			method:      http.MethodPost,
			handlerFunc: s.serveCreateUser,
		},
		// Edit a user
		{
			path:        "/api/user",
			method:      http.MethodPatch,
			handlerFunc: s.serveUpdateUser,
		},
		// Load user.
		{
			path:        "/api/user",
			method:      http.MethodGet,
			handlerFunc: s.serveUser,
		},
		// New game.
		{
			path:        "/api/game",
			method:      http.MethodPost,
			handlerFunc: s.serveCreateGame,
		},
		// Pending games.
		{
			path:        "/api/games",
			method:      http.MethodGet,
			handlerFunc: s.servePendingGames,
		},
		// Get game.
		{
			path:        "/api/game/{id}",
			method:      http.MethodGet,
			handlerFunc: s.requireGameAuth(s.serveGame),
		},
		// Get players.
		{
			path:        "/api/game/{id}/players",
			method:      http.MethodGet,
			handlerFunc: s.requireGameAuth(s.serveGamePlayers),
		},
		// Join game.
		{
			path:        "/api/game/{id}/join",
			method:      http.MethodPost,
			handlerFunc: s.requireGameAuth(s.serveJoinGame, isGamePending()),
		},
		// Assign roles
		{
			path:        "/api/game/{id}/assignRole",
			method:      http.MethodPost,
			handlerFunc: s.requireGameAuth(s.serveAssignRole, isGameCreator(), isGamePending()),
		},
		// Start game.
		{
			path:        "/api/game/{id}/start",
			method:      http.MethodPost,
			handlerFunc: s.requireGameAuth(s.serveStartGame, isGameCreator(), isGamePending()),
		},
		// Serve a clue to a game.
		{
			path:        "/api/game/{id}/clue",
			method:      http.MethodPost,
			handlerFunc: s.requireGameAuth(s.serveClue, isSpymaster()),
		},
		// Serve a card guess to a game.
		{
			path:        "/api/game/{id}/guess",
			method:      http.MethodPost,
			handlerFunc: s.requireGameAuth(s.serveGuess, isOperative()),
		},
		// WebSocket handler for games.
		{
			path:        "/api/game/{id}/ws",
			method:      http.MethodGet,
			handlerFunc: s.requireGameAuth(s.serveData),
		},
	}

	for _, h := range handlers {
		m.HandleFunc(h.path, s.handleError(h.handlerFunc)).Methods(h.method)
	}

	return m
}

func (s *Srv) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Srv) handleError(h handlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := h(w, r)
		if err == nil {
			return
		}
		log.Println(err)

		code, userMsg := httperr.Extract(err)
		http.Error(w, userMsg, code)
	}
}

func (s *Srv) serveCreateUser(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return httperr.BadRequest("failed to decode create user request: %w", err)
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return httperr.
			BadRequest("create user request contained no name").
			WithMessage("no name given")
	}

	id, err := s.db.NewUser(&codenames.User{Name: name})
	if err != nil {
		return httperr.
			Internal("failed to create user with name %q: %w", name, err).
			WithMessage("failed to create user")
	}

	encoded, err := s.sc.Encode("auth", id)
	if err != nil {
		return httperr.
			Internal("failed to encode auth for id %q: %w", id, err).
			WithMessage("failed to encode credentials")
	}

	http.SetCookie(w, &http.Cookie{
		Name:  "Authorization",
		Value: encoded,
	})

	return jsonResp(w, struct {
		UserID  string `json:"user_id"`
		Success bool   `json:"success"`
	}{string(id), true})
}

func (s *Srv) serveUpdateUser(w http.ResponseWriter, r *http.Request) error {
	u, err := s.loadUser(r)
	if err != nil {
		return err
	}

	// TODO(bcspragu): Allow updating the user.

	return jsonResp(w, u)
}

func (s *Srv) serveUser(w http.ResponseWriter, r *http.Request) error {
	u, err := s.loadUser(r)
	if err != nil {
		return err
	}

	return jsonResp(w, u)
}

func (s *Srv) serveCreateGame(w http.ResponseWriter, r *http.Request) error {
	u, err := s.loadUserRequired(r)
	if err != nil {
		return err
	}

	ar := codenames.RedTeam
	if s.r.Intn(2) == 0 {
		ar = codenames.BlueTeam
	}

	id, err := s.db.NewGame(&codenames.Game{
		CreatedBy: u.ID,
		State: &codenames.GameState{
			StartingTeam: ar,
			ActiveTeam:   ar,
			ActiveRole:   codenames.SpymasterRole,
			Board:        boardgen.New(ar, s.r),
		},
	})
	if err != nil {
		return httperr.
			Internal("failed to create game for user %q: %w", u.ID, err).
			WithMessage("failed to create game")
	}

	return jsonResp(w, struct {
		ID string `json:"id"`
	}{string(id)})
}

func (s *Srv) servePendingGames(w http.ResponseWriter, r *http.Request) error {
	gIDs, err := s.db.PendingGames()
	if err != nil {
		return httperr.Internal("failed to load pending games: %w", err)
	}

	return jsonResp(w, gIDs)
}

func (s *Srv) serveGame(w http.ResponseWriter, r *http.Request, u *codenames.User, game *codenames.Game, userPR *codenames.PlayerRole, prs []*codenames.PlayerRole) error {
	// If you aren't in this game or ain't a spymaster, you don't get to see what
	// color all the cards are, that's [REDACTED].
	if userPR == nil || userPR.Role != codenames.SpymasterRole {
		game.State.Board = codenames.Revealed(game.State.Board)
	}

	return jsonResp(w, game)
}

func (s *Srv) serveGamePlayers(w http.ResponseWriter, r *http.Request, u *codenames.User, game *codenames.Game, userPR *codenames.PlayerRole, prs []*codenames.PlayerRole) error {
	players, err := s.toPlayers(prs)
	if err != nil {
		return httperr.
			Internal("failed to convert players in game %q: %w", game.ID, err).
			WithMessage("failed to make players")
	}

	return jsonResp(w, players)
}

func (s *Srv) serveJoinGame(w http.ResponseWriter, r *http.Request, u *codenames.User, game *codenames.Game, userPR *codenames.PlayerRole, prs []*codenames.PlayerRole) error {
	if userPR != nil {
		// They've already joined the game, just return success because we'd
		// probably fail trying to add them again.
		return jsonResp(w, struct {
			Success bool `json:"success"`
		}{true})
	}

	var req struct {
		PlayerType string `json:"player_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return httperr.BadRequest("failed to decode join request: %w", err)
	}

	playerType, ok := codenames.ToPlayerType(req.PlayerType)
	if !ok {
		return httperr.
			BadRequest("unknown player type %q given", req.PlayerType).
			WithMessage("bad player type")
	}

	pID := codenames.PlayerID{
		PlayerType: playerType,
		ID:         string(u.ID),
	}
	if err := s.db.JoinGame(game.ID, pID); err != nil {
		return httperr.
			Internal("failed to join game %q with user %q: %w", game.ID, u.ID, err).
			WithMessage("failed to join game")
	}

	return jsonResp(w, struct {
		Success bool `json:"success"`
	}{true})
}

func (s *Srv) serveAssignRole(w http.ResponseWriter, r *http.Request, creator *codenames.User, game *codenames.Game, userPR *codenames.PlayerRole, prs []*codenames.PlayerRole) error {
	var req struct {
		PlayerID codenames.PlayerID `json:"player_id"`
		Team     string             `json:"team"`
		Role     string             `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return httperr.BadRequest("failed to decode assign role request: %w", err)
	}

	if _, err := s.db.Player(req.PlayerID); err != nil {
		return httperr.
			BadRequest("failed to load player %q in assignRole: %w", req.PlayerID, err).
			WithMessage("bad player ID given")
	}
	pID := req.PlayerID

	desiredRole, ok := codenames.ToRole(req.Role)
	if !ok {
		return httperr.
			BadRequest("unknown role %q given", req.Role).
			WithMessage("bad role")
	}

	desiredTeam, ok := codenames.ToTeam(req.Team)
	if !ok {
		return httperr.
			BadRequest("unknown team %q given", req.Team).
			WithMessage("bad team")
	}

	roleCount := make(map[codenames.Role]map[codenames.Team]int)
	for _, pr := range prs {
		if !pr.RoleAssigned {
			continue
		}

		rc, ok := roleCount[pr.Role]
		if !ok {
			rc = make(map[codenames.Team]int)
		}
		if pr.PlayerID == pID {
			return httperr.
				BadRequest("player %q tried to join game %q as %q %q, already joined as %q %q", pID, game.ID, desiredTeam, desiredRole, pr.Team, pr.Role).
				WithMessage(fmt.Sprintf("can't join game as %q %q, already joined as %q %q", desiredTeam, desiredRole, pr.Team, pr.Role))
		}
		if pr.Role == codenames.SpymasterRole && rc[pr.Team] > 1 {
			return httperr.
				Internal("game %q in bad state, has multiple players has %q spymaster", game.ID, pr.Team).
				WithMessage(fmt.Sprintf("multiple players set as %q spymaster", pr.Team))
		}
		if pr.Role == codenames.OperativeRole && rc[pr.Team] > maxOperativesPerTeam {
			return httperr.
				Internal("game %q in bad state, has too many players as %q operatives", game.ID, pr.Team).
				WithMessage(fmt.Sprintf("too many players set as %q operatives", pr.Team))
		}
		rc[pr.Team]++
		roleCount[pr.Role] = rc
	}

	if desiredRole == codenames.SpymasterRole && roleCount[codenames.SpymasterRole][desiredTeam] > 0 {
		return httperr.
			BadRequest("player %q wanted to be %q spymaster, but that role is already filled in game %q", pID, desiredTeam, game.ID).
			WithMessage(fmt.Sprintf("team %q already has a spymaster", desiredTeam))
	}
	if desiredRole == codenames.OperativeRole && roleCount[codenames.OperativeRole][desiredTeam] >= maxOperativesPerTeam {
		return httperr.
			BadRequest("player %q wanted to be a %q operative, but that team already has the max number of operatives in game %q", pID, desiredTeam, game.ID).
			WithMessage(fmt.Sprintf("team %q already has max operatives", desiredTeam))
	}

	if err := s.db.AssignRole(game.ID, &codenames.PlayerRole{
		PlayerID: pID,
		Team:     desiredTeam,
		Role:     desiredRole,
	}); err != nil {
		return httperr.
			Internal("failed to assign role (%q, %q) to player %q in game %q: %w", desiredTeam, desiredRole, pID, game.ID, err).
			WithMessage("failed to assign role to player")
	}

	// Load the updated list of players in the game.
	prs, err := s.db.PlayersInGame(game.ID)
	if err != nil {
		return httperr.
			Internal("failed to load players in game %q: %w", game.ID, err).
			WithMessage("failed to load players in game")
	}

	players, err := s.toPlayers(prs)
	if err != nil {
		return httperr.
			Internal("failed to convert players in game %q: %w", game.ID, err).
			WithMessage("failed to make players")
	}

	return jsonResp(w, players)
}

func (s *Srv) serveStartGame(w http.ResponseWriter, r *http.Request, u *codenames.User, game *codenames.Game, userPR *codenames.PlayerRole, prs []*codenames.PlayerRole) error {
	var req struct {
		RandomAssignment bool `json:"random_assignment"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return httperr.BadRequest("failed to decode start game request: %w", err)
	}

	if req.RandomAssignment {
		modified, err := s.finishAssigningRoles(game, prs)
		if err != nil {
			return err
		}
		if modified {
			// Load the player roles again since we modified them during random
			// assignment.
			newPRS, err := s.db.PlayersInGame(game.ID)
			if err != nil {
				return httperr.
					Internal("failed to load players in game %q: %w", game.ID, err).
					WithMessage("failed to load players in game")
			}
			prs = newPRS
		}
	}

	// Now, check that we don't have any unassigned folks in the lobby.
	for _, pr := range prs {
		if !pr.RoleAssigned {
			return httperr.
				BadRequest("can't start game because player %+v hasn't been given a role", pr.PlayerID).
				WithMessage("at least one player hasn't been assigned a role")
		}
	}

	if err := codenames.AllRolesFilled(prs); err != nil {
		return httperr.
			BadRequest("user %q tried to start game %q, but not all roles are filled: %w", u.ID, game.ID, err).
			WithMessage(fmt.Sprintf("can't start game yet: %v", err))
	}

	// If we're here, all the right roles are filled, the game is pending, and
	// the caller is the one who created the game, let's start it.
	if err := s.db.StartGame(game.ID); err != nil {
		return httperr.
			Internal("failed to start game %q: %w", game.ID, err).
			WithMessage("failed to start game")
	}
	game.Status = codenames.Playing

	players, err := s.toPlayers(prs)
	if err != nil {
		return httperr.
			Internal("failed to convert players in game %q: %w", game.ID, err).
			WithMessage("failed to make players")
	}

	if err := s.broadcastMessage(game, prs, func(g *codenames.Game) interface{} {
		return &GameStart{
			Game:    g,
			Players: players,
		}
	}); err != nil {
		return httperr.
			Internal("failed to send start for game %q: %w", game.ID, err).
			WithMessage("failed to send game start message")
	}

	return jsonResp(w, struct {
		Success bool `json:"success"`
	}{true})
}

func (s *Srv) finishAssigningRoles(game *codenames.Game, prs []*codenames.PlayerRole) (bool, error) {
	if len(prs) < 4 {
		return false, httperr.
			BadRequest("only have %d players, need four to start a game", len(prs)).
			WithMessage("you need at least four players to start")
	}

	// Start by marking both spymaster positions available.
	availableSpymasterPos := map[codenames.Team]bool{
		codenames.BlueTeam: true,
		codenames.RedTeam:  true,
	}

	// Now, find all the users without roles, and mark taking roles as such.
	var unassigned []*codenames.PlayerRole
	for _, pr := range prs {
		if !pr.RoleAssigned {
			unassigned = append(unassigned, pr)
			continue
		}

		// Only spymasters get marked as 'taken', since we can have any number of
		// operatives.
		if pr.Role == codenames.SpymasterRole {
			availableSpymasterPos[pr.Team] = false
		}
	}

	// Now, shuffle the unassigned users.
	s.r.Shuffle(len(unassigned), func(i, j int) {
		unassigned[i], unassigned[j] = unassigned[j], unassigned[i]
	})

	attemptAssignSpymaster := func(pr *codenames.PlayerRole) (bool, error) {
		for _, team := range []codenames.Team{codenames.RedTeam, codenames.BlueTeam} {
			if !availableSpymasterPos[team] {
				continue
			}

			// Assign this player to the spymaster role, since it's available.
			if err := s.db.AssignRole(game.ID, &codenames.PlayerRole{
				PlayerID: pr.PlayerID,
				Team:     team,
				Role:     codenames.SpymasterRole,
			}); err != nil {
				return false, httperr.
					Internal("failed to assign %+v to %s spymaster: %w", pr.PlayerID, team, err).
					WithMessage("failed to randomly assign players")
			}
			availableSpymasterPos[team] = false
			return true, nil
		}
		return false, nil
	}

	for i, pr := range unassigned {
		// First, try to assign to spymaster roles.
		assigned, err := attemptAssignSpymaster(pr)
		if err != nil {
			return false, err
		}
		if assigned {
			continue
		}

		// If there are no spymaster positions open, pick an operative team and
		// assign them.
		team := codenames.RedTeam
		if i%2 == 1 {
			team = codenames.BlueTeam
		}

		if err := s.db.AssignRole(game.ID, &codenames.PlayerRole{
			PlayerID: pr.PlayerID,
			Team:     team,
			Role:     codenames.OperativeRole,
		}); err != nil {
			return false, httperr.
				Internal("failed to assign %+v to %s operative: %w", pr.PlayerID, team, err).
				WithMessage("failed to randomly assign players")
		}
	}

	return len(unassigned) > 0, nil
}

func (s *Srv) toPlayers(prs []*codenames.PlayerRole) ([]*Player, error) {
	var ids []codenames.PlayerID
	for _, pr := range prs {
		ids = append(ids, pr.PlayerID)
	}

	names, err := s.db.BatchPlayerNames(ids)
	if err != nil {
		return nil, fmt.Errorf("failed to load player names: %w", err)
	}

	var out []*Player
	for _, pr := range prs {
		name, ok := names[pr.PlayerID]
		if !ok {
			return nil, fmt.Errorf("no name was returned for player ID %q", pr.PlayerID)
		}
		out = append(out, &Player{
			PlayerID: pr.PlayerID,
			Name:     name,
			Team:     pr.Team,
			Role:     pr.Role,
		})
	}

	return out, nil
}

func (s *Srv) broadcastMessage(game *codenames.Game, prs []*codenames.PlayerRole, fn func(*codenames.Game) interface{}) error {
	// First, send the full board to the spymasters.
	fullMsg := fn(game)
	for _, pr := range prs {
		if pr.Role != codenames.SpymasterRole {
			continue
		}
		if pr.PlayerID.PlayerType != codenames.PlayerTypeHuman {
			continue
		}

		uID := codenames.UserID(pr.PlayerID.ID)
		if err := s.hub.ToUser(game.ID, uID, fullMsg); err != nil {
			return fmt.Errorf("failed to send spymaster msg: %w", err)
		}
	}

	// Now, clear out the card agent colorings and send that board to everyone
	// else.
	game.State.Board = codenames.Revealed(game.State.Board)
	operativeMsg := fn(game)
	for _, pr := range prs {
		if pr.Role != codenames.OperativeRole {
			continue
		}
		if pr.PlayerID.PlayerType != codenames.PlayerTypeHuman {
			continue
		}

		uID := codenames.UserID(pr.PlayerID.ID)
		if err := s.hub.ToUser(game.ID, uID, operativeMsg); err != nil {
			return fmt.Errorf("failed to send operative msg: %w", err)
		}
	}

	return nil
}

func (s *Srv) serveClue(w http.ResponseWriter, r *http.Request, u *codenames.User, g *codenames.Game, userPR *codenames.PlayerRole, prs []*codenames.PlayerRole) error {
	if g.Status != codenames.Playing {
		return httperr.
			BadRequest("player %q tried to give clue in game %q, which is in state %q", u.ID, g.ID, g.Status).
			WithMessage("can't give clues to a not-playing game")
	}

	var req struct {
		Word  string `json:"word"`
		Count int    `json:"count"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return httperr.BadRequest("failed to decode give clue request: %w", err)
	}

	clue := &codenames.Clue{
		Word:  req.Word,
		Count: req.Count,
	}
	// We don't need to check if the status changed/game is over, because giving
	// a clue will never end the game.
	newState, newStatus, err := game.NewForMove(g.State).Move(&game.Move{
		Action:   game.ActionGiveClue,
		Team:     userPR.Team,
		GiveClue: clue,
	})
	if err != nil {
		// We assume the error is the result of a bad request.
		return httperr.
			BadRequest("player %q in game %q gave invalid clue: %w", u.ID, g.ID, err).
			WithMessage(fmt.Sprintf("failed to make move: %v", err))
	}

	// Update the state in the database.
	if err := s.db.UpdateState(g.ID, newState); err != nil {
		return httperr.
			Internal("failed to update state for game %q: %w", g.ID, err).
			WithMessage("failed to update game state")
	}
	g.State = newState
	g.Status = newStatus

	// Send the clue down to everyone.
	if err := s.broadcastMessage(g, prs, func(g *codenames.Game) interface{} {
		return &ClueGiven{
			Clue: clue,
			Team: userPR.Team,
			Game: g,
		}
	}); err != nil {
		return httperr.
			Internal("failed to send clue for game %q: %w", g.ID, err).
			WithMessage("failed to inform players of clue")
	}

	return jsonResp(w, struct {
		Success bool `json:"success"`
	}{true})
}

func (s *Srv) serveGuess(w http.ResponseWriter, r *http.Request, u *codenames.User, g *codenames.Game, userPR *codenames.PlayerRole, prs []*codenames.PlayerRole) error {
	if g.Status != codenames.Playing {
		return httperr.
			BadRequest("player %q tried to guess in game %q, which is in state %q", u.ID, g.ID, g.Status).
			WithMessage("can't guess in a not-playing game")
	}

	// Since we record votes and calculate consensus before making the move, we
	// need to independently validate moves first.
	if userPR.Team != g.State.ActiveTeam {
		return httperr.
			BadRequest("user %q of team %q tried to guess when %q %q was active in game %q", u.ID, userPR.Team, g.State.ActiveTeam, g.State.ActiveRole, g.ID).
			WithMessage("it's not your team's turn")
	}

	if g.State.ActiveRole != codenames.OperativeRole {
		return httperr.
			BadRequest("user %q as %q %q tried to guess when %q %q was active in game %q", u.ID, userPR.Team, userPR.Role, g.State.ActiveTeam, g.State.ActiveRole, g.ID).
			WithMessage("it's not time to guess")
	}

	var req struct {
		Guess     string `json:"guess"`
		Confirmed bool   `json:"confirmed"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return httperr.BadRequest("failed to decode guess request: %w", err)
	}

	card, ok := findCard(g.State.Board.Cards, req.Guess)
	if !ok {
		return httperr.
			BadRequest("user %q guessed %q, which didn't correspond to a card in game %q", u.ID, req.Guess, g.ID).
			WithMessage(fmt.Sprintf("guess %q didn't correspond to a card", req.Guess))
	}

	if err := s.hub.ToGame(g.ID, &PlayerVote{
		UserID:    u.ID,
		Guess:     req.Guess,
		Confirmed: req.Confirmed,
	}); err != nil {
		return httperr.
			Internal("failed to send player vote for game %q: %w", g.ID, err).
			WithMessage("failed to inform players of vote")
	}

	if !req.Confirmed {
		// If it's not confirmed (e.g. it's just tentative), so we shouldn't count
		// the votes.
		return nil
	}

	guess, hasConsensus := s.consensus.RecordVote(g.ID, u.ID, req.Guess, countVoters(prs, g.State.ActiveTeam))
	if !hasConsensus {
		return nil
	}

	if card, ok = findCard(g.State.Board.Cards, guess); !ok {
		// This should probably never happen because if we have consensus, it
		// should be *this* vote that caused it, but because HTTP requests are
		// asynchronous, it could theoretically happen.
		return httperr.
			BadRequest("team %q guessed %q, which didn't correspond to a card in game %q", userPR.Team, req.Guess, g.ID).
			WithMessage(fmt.Sprintf("guess %q didn't correspond to a card", guess))
	}

	gfm := game.NewForMove(g.State)

	newState, newStatus, err := gfm.Move(&game.Move{
		Action: game.ActionGuess,
		Team:   userPR.Team,
		Guess:  guess,
	})
	if err != nil {
		// We assume the error is the result of a bad request.
		return httperr.
			BadRequest("player %q/team %q in game %q gave invalid guess: %w", u.ID, userPR.Team, g.ID, err).
			WithMessage(fmt.Sprintf("failed to make move: %v", err))
	}

	if card, ok = findCard(newState.Board.Cards, guess); !ok {
		return httperr.
			Internal("guess %q somehow no longer exists in the cards of game %q", guess, g.ID).
			WithMessage(fmt.Sprintf("guess %q didn't correspond to a card", guess))
	}

	// They've made the guess, clear out the consensus for the next time.
	s.consensus.Clear(g.ID)

	// Update the state in the database.
	if err := s.db.UpdateState(g.ID, newState); err != nil {
		return httperr.
			Internal("failed to update state for game %q: %w", g.ID, err).
			WithMessage("failed to update game state")
	}

	// Players can keep guessing if the game tells us its still their turn.
	canKeepGuessing := newState.ActiveRole == codenames.OperativeRole && newStatus != codenames.Finished
	if err := s.broadcastMessage(g, prs, func(g *codenames.Game) interface{} {
		return &GuessGiven{
			Guess:           guess,
			Team:            userPR.Team,
			CanKeepGuessing: canKeepGuessing,
			RevealedCard:    card,
			Game:            g,
		}
	}); err != nil {
		return httperr.
			Internal("failed to send guess for game %q: %w", g.ID, err).
			WithMessage("failed to inform players of guess")
	}

	if newStatus != codenames.Finished {
		return nil
	}

	over, winningTeam := gfm.GameOver()
	if !over {
		return httperr.
			Internal("state for game %q was finished, but GameOver() says the game isn't over", g.ID).
			WithMessage("error with game state")
	}

	// The game is over, we should let folks know.
	if err := s.hub.ToGame(g.ID, &GameEnd{
		WinningTeam: winningTeam,
	}); err != nil {
		return httperr.
			Internal("failed to send game over for game %q: %w", g.ID, err).
			WithMessage("failed to inform players of game over")
	}

	return jsonResp(w, struct {
		Success bool `json:"success"`
	}{true})
}

func countVoters(prs []*codenames.PlayerRole, team codenames.Team) int {
	cnt := 0
	for _, pr := range prs {
		if pr.Role == codenames.OperativeRole && pr.Team == team {
			cnt++
		}
	}
	return cnt
}

func (s *Srv) serveData(w http.ResponseWriter, r *http.Request, u *codenames.User, game *codenames.Game, userPR *codenames.PlayerRole, prs []*codenames.PlayerRole) error {
	conn, err := s.ws.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return httperr.
			Internal("failed to upgrade connection for user %q, game %q: %w", u.ID, game.ID, err).
			WithMessage("failed to connect")
	}

	s.hub.Register(conn, game.ID, u.ID)

	return nil
}

func jsonResp(w http.ResponseWriter, v interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return httperr.Internal("failed to encode response for %+v of type %T: %w", v, v, err)
	}

	return nil
}

func (s *Srv) loadUserRequired(r *http.Request) (*codenames.User, error) {
	u, err := s.loadUser(r)
	if err != nil {
		return nil, err
	}

	if u == nil {
		return nil, httperr.
			Unauthorized("no user in request for %q", r.URL.Path).
			WithMessage("no valid user auth provided")
	}

	return u, nil
}

func (s *Srv) loadUser(r *http.Request) (*codenames.User, error) {
	c, err := r.Cookie("Authorization")
	if err == http.ErrNoCookie {
		return nil, nil
	}
	if err != nil {
		return nil, httperr.
			Internal("failed to load auth cookie: %w", err).
			WithMessage("failed to load user")
	}

	var uID codenames.UserID
	if err := s.sc.Decode("auth", c.Value, &uID); err != nil {
		// If we can't parse it, assume it's an old auth cookie and treat them as
		// not logged in.
		return nil, nil
	}

	u, err := s.db.User(uID)
	if err == codenames.ErrUserNotFound {
		// Same deal here. If they have a valid cookie but we can't find the user,
		// assume we wiped the DB or something and treat them as not logged in.
		return nil, nil
	} else if err != nil {
		return nil, httperr.
			Internal("failed to load user from DB: %w", err).
			WithMessage("failed to load user")
	}

	return u, nil
}

type gameHandler func(w http.ResponseWriter, r *http.Request, u *codenames.User, game *codenames.Game, userPR *codenames.PlayerRole, prs []*codenames.PlayerRole) error

type gameAuthOption func(*gameAuthOptions)

func isGameCreator() gameAuthOption {
	return func(opts *gameAuthOptions) {
		opts.isGameCreator = true
	}
}

func isGamePending() gameAuthOption {
	return isGameStatus(codenames.Pending)
}

func isGameStatus(gs codenames.GameStatus) gameAuthOption {
	return func(opts *gameAuthOptions) {
		opts.wantGameStatus = gs
	}
}

func isSpymaster() gameAuthOption {
	return isRole(codenames.SpymasterRole)
}

func isOperative() gameAuthOption {
	return isRole(codenames.OperativeRole)
}

func isRole(r codenames.Role) gameAuthOption {
	return func(opts *gameAuthOptions) {
		opts.wantRole = r
	}
}

type gameAuthOptions struct {
	isGameCreator  bool
	wantRole       codenames.Role
	wantGameStatus codenames.GameStatus
}

func (s *Srv) requireGameAuth(handler gameHandler, opts ...gameAuthOption) handlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		gOpts := &gameAuthOptions{}
		for _, opt := range opts {
			opt(gOpts)
		}

		gID, err := s.gameIDFromRequest(r)
		if err != nil {
			return err
		}

		u, err := s.loadUserRequired(r)
		if err != nil {
			return err
		}

		game, err := s.db.Game(gID)
		if err != nil {
			return httperr.
				Internal("failed to load game %q: %w", gID, err).
				WithMessage("failed to load game")
		}

		if gOpts.isGameCreator && game.CreatedBy != u.ID {
			return httperr.
				Forbidden("user %q tried to do an admin action on game %q, which was created by %q", u.ID, game.ID, game.CreatedBy).
				WithMessage("only the game creator can perform this action")
		}

		if gOpts.wantGameStatus != codenames.NoStatus && gOpts.wantGameStatus != game.Status {
			return httperr.
				BadRequest("user %q tried to act on game %q in state %q, can only act if state %q", u.ID, gID, game.Status, gOpts.wantGameStatus).
				WithMessage("the game isn't in a state where you can do that")
		}

		prs, err := s.db.PlayersInGame(gID)
		if err != nil {
			return httperr.
				Internal("failed to load players in game %q: %w", gID, err).
				WithMessage("failed to load players in game")
		}

		userPR, ok := findRole(u.ID, prs)
		if !ok && gOpts.wantRole != codenames.NoRole {
			return httperr.
				Forbidden("user %q is not in game %q", u.ID, gID).
				WithMessage("you need to join this game first")
		}

		return handler(w, r, u, game, userPR, prs)
	}
}

func findRole(uID codenames.UserID, prs []*codenames.PlayerRole) (*codenames.PlayerRole, bool) {
	for _, pr := range prs {
		if pr.PlayerID.PlayerType != codenames.PlayerTypeHuman {
			continue
		}

		if pr.PlayerID.ID == string(uID) {
			return pr, true
		}
	}
	return nil, false
}

func (s *Srv) gameIDFromRequest(r *http.Request) (codenames.GameID, error) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		return "", httperr.
			BadRequest("no game ID provided for request to %q", r.URL.Path).
			WithMessage("no game ID provided")
	}
	return codenames.GameID(id), nil
}

func findCard(cards []codenames.Card, guess string) (*codenames.Card, bool) {
	guess = strings.ToLower(guess)
	for _, c := range cards {
		if strings.ToLower(c.Codename) == guess {
			return &c, true
		}
	}
	return nil, false
}
