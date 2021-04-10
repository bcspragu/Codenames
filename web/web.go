package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strings"

	"github.com/bcspragu/Codenames/boardgen"
	"github.com/bcspragu/Codenames/codenames"
	"github.com/bcspragu/Codenames/consensus"
	"github.com/bcspragu/Codenames/game"
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
func New(db codenames.DB, r *rand.Rand) (*Srv, error) {
	sc, err := loadKeys()
	if err != nil {
		return nil, err
	}

	s := &Srv{
		sc:        sc,
		hub:       hub.New(),
		db:        db,
		r:         r,
		ws:        &websocket.Upgrader{}, // use default options, for now
		consensus: consensus.New(),
	}

	s.mux = s.initMux()

	return s, nil
}

func (s *Srv) initMux() *mux.Router {
	m := mux.NewRouter()
	// New user.
	m.HandleFunc("/api/user", s.serveCreateUser).Methods("POST")
	// Load user.
	m.HandleFunc("/api/user", s.serveUser).Methods("GET")
	// New game.
	m.HandleFunc("/api/game", s.serveCreateGame).Methods("POST")
	// Pending games.
	m.HandleFunc("/api/games", s.servePendingGames).Methods("GET")
	// Get game.
	m.HandleFunc("/api/game/{id}", s.requireGameAuth(s.serveGame)).Methods("GET")
	// Join game.
	m.HandleFunc("/api/game/{id}/join", s.serveJoinGame).Methods("POST")
	// Start game.
	m.HandleFunc("/api/game/{id}/start", s.requireGameAuth(s.serveStartGame)).Methods("POST")
	// Serve a clue to a game.
	m.HandleFunc("/api/game/{id}/clue", s.requireGameAuth(s.serveClue)).Methods("POST")
	// Serve a card guess to a game.
	m.HandleFunc("/api/game/{id}/guess", s.requireGameAuth(s.serveGuess)).Methods("POST")

	// WebSocket handler for games.
	m.HandleFunc("/api/game/{id}/ws", s.requireGameAuth(s.serveData)).Methods("GET")

	return m
}

func (s *Srv) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Srv) serveCreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		http.Error(w, "No name given", http.StatusBadRequest)
		return
	}

	id, err := s.db.NewUser(&codenames.User{Name: name})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	encoded, err := s.sc.Encode("auth", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:  "Authorization",
		Value: encoded,
	})

	jsonResp(w, struct {
		Success bool `json:"success"`
	}{true})
}

func (s *Srv) serveUser(w http.ResponseWriter, r *http.Request) {
	u, err := s.loadUser(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResp(w, u)
}

func (s *Srv) serveCreateGame(w http.ResponseWriter, r *http.Request) {
	u, err := s.loadUser(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if u == nil {
		http.Error(w, "Not logged in", http.StatusUnauthorized)
		return
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResp(w, struct {
		ID string `json:"id"`
	}{string(id)})
}

func (s *Srv) servePendingGames(w http.ResponseWriter, r *http.Request) {
	gIDs, err := s.db.PendingGames()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResp(w, gIDs)
}

func (s *Srv) serveGame(w http.ResponseWriter, r *http.Request, u *codenames.User, game *codenames.Game, userPR *codenames.PlayerRole, prs []*codenames.PlayerRole) {
	if userPR.Role != codenames.SpymasterRole {
		game.State.Board = codenames.Revealed(game.State.Board)
	}

	jsonResp(w, game)
}

func (s *Srv) serveJoinGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, ok := vars["id"]
	if !ok {
		http.Error(w, "no game ID provided", http.StatusBadRequest)
		return
	}
	gID := codenames.GameID(id)

	var req struct {
		Team string `json:"team"`
		Role string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	desiredRole, ok := codenames.ToRole(req.Role)
	if !ok {
		http.Error(w, "bad role", http.StatusBadRequest)
		return
	}

	desiredTeam, ok := codenames.ToTeam(req.Team)
	if !ok {
		http.Error(w, "bad team", http.StatusBadRequest)
		return
	}

	u, err := s.loadUser(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if u == nil {
		http.Error(w, "Not logged in", http.StatusUnauthorized)
		return
	}

	game, err := s.db.Game(gID)
	if err != nil {
		http.Error(w, "failed to load game", http.StatusInternalServerError)
		return
	}

	if game.Status != codenames.Pending {
		http.Error(w, "can only join pending games", http.StatusBadRequest)
		return
	}

	prs, err := s.db.PlayersInGame(gID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	roleCount := make(map[codenames.Role]map[codenames.Team]int)
	for _, pr := range prs {
		rc, ok := roleCount[pr.Role]
		if !ok {
			rc = make(map[codenames.Team]int)
		}
		if pr.PlayerID.IsUser(u.ID) {
			errMsg := fmt.Sprintf("can't join game as %q %q, already joined as %q %q", desiredTeam, desiredRole, pr.Team, pr.Role)
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		if pr.Role == codenames.SpymasterRole && rc[pr.Team] > 1 {
			http.Error(w, fmt.Sprintf("multiple players set as %q spymaster", pr.Team), http.StatusInternalServerError)
			return
		}
		if pr.Role == codenames.OperativeRole && rc[pr.Team] > maxOperativesPerTeam {
			http.Error(w, fmt.Sprintf("too many players set as %q operatives", pr.Team), http.StatusInternalServerError)
			return
		}
		rc[pr.Team]++
		roleCount[pr.Role] = rc
	}

	if desiredRole == codenames.SpymasterRole && roleCount[codenames.SpymasterRole][desiredTeam] > 0 {
		http.Error(w, fmt.Sprintf("team %q already has a spymaster", desiredTeam), http.StatusBadRequest)
		return
	}
	if desiredRole == codenames.OperativeRole && roleCount[codenames.OperativeRole][desiredTeam] >= maxOperativesPerTeam {
		http.Error(w, fmt.Sprintf("team %q already has max operatives", desiredTeam), http.StatusBadRequest)
		return
	}

	if err := s.db.JoinGame(gID, &codenames.PlayerRole{
		PlayerID: codenames.PlayerID{
			PlayerType: codenames.PlayerTypeHuman,
			ID:         string(u.ID),
		},
		Team: desiredTeam,
		Role: desiredRole,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResp(w, struct {
		Success bool `json:"success"`
	}{true})
}

func (s *Srv) serveStartGame(w http.ResponseWriter, r *http.Request, u *codenames.User, game *codenames.Game, userPR *codenames.PlayerRole, prs []*codenames.PlayerRole) {
	if game.CreatedBy != u.ID {
		http.Error(w, "only the game creator can start the game", http.StatusForbidden)
		return
	}

	if game.Status != codenames.Pending {
		http.Error(w, "can only start pending games", http.StatusBadRequest)
		return
	}

	if err := codenames.AllRolesFilled(prs); err != nil {
		http.Error(w, fmt.Sprintf("can't start game yet: %v", err), http.StatusBadRequest)
		return
	}

	// If we're here, all the right roles are filled, the game is pending, and
	// the caller is the one who created the game, let's start it.
	if err := s.db.StartGame(game.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	game.Status = codenames.Playing

	players, err := s.toPlayers(prs)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to make players: %v", err), http.StatusInternalServerError)
		return
	}

	if err := s.broadcastMessage(game, prs, func(g *codenames.Game) interface{} {
		return &GameStart{
			Game:    g,
			Players: players,
		}
	}); err != nil {
		http.Error(w, fmt.Sprintf("failed to send game start: %v", err), http.StatusInternalServerError)
		return
	}

	jsonResp(w, struct {
		Success bool `json:"success"`
	}{true})
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

func (s *Srv) serveClue(w http.ResponseWriter, r *http.Request, u *codenames.User, g *codenames.Game, userPR *codenames.PlayerRole, prs []*codenames.PlayerRole) {
	if g.Status != codenames.Playing {
		http.Error(w, "can't give clues to a not-playing game", http.StatusBadRequest)
		return
	}

	var req struct {
		Word  string `json:"word"`
		Count int    `json:"count"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	clue := &codenames.Clue{
		Word:  req.Word,
		Count: req.Count,
	}
	// We don't need to check if the status changed/game is over, because giving
	// a clue will never end the game.
	newState, _, err := game.NewForMove(g.State).Move(&game.Move{
		Action:   game.ActionGiveClue,
		Team:     userPR.Team,
		GiveClue: clue,
	})
	if err != nil {
		// We assume the error is the result of a bad request.
		http.Error(w, fmt.Sprintf("failed to make move : %v", err), http.StatusBadRequest)
		return
	}

	// Update the state in the database.
	if err := s.db.UpdateState(g.ID, newState); err != nil {
		http.Error(w, fmt.Sprintf("failed to update game state: %v", err), http.StatusInternalServerError)
		return
	}

	// Send the clue down to everyone.
	if err := s.hub.ToGame(g.ID, &ClueGiven{
		Clue: clue,
		Team: userPR.Team,
	}); err != nil {
		http.Error(w, fmt.Sprintf("failed to inform players of clue: %v", err), http.StatusInternalServerError)
		return
	}
}

func (s *Srv) serveGuess(w http.ResponseWriter, r *http.Request, u *codenames.User, g *codenames.Game, userPR *codenames.PlayerRole, prs []*codenames.PlayerRole) {
	if g.Status != codenames.Playing {
		http.Error(w, "can't guess in a not-playing game", http.StatusBadRequest)
		return
	}

	if userPR.Team != g.State.ActiveTeam {
		http.Error(w, "it's not your team's turn", http.StatusBadRequest)
		return
	}

	if g.State.ActiveRole != codenames.OperativeRole {
		http.Error(w, "it's not time to guess", http.StatusBadRequest)
		return
	}

	var req struct {
		Guess     string `json:"guess"`
		Confirmed bool   `json:"confirmed"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.hub.ToGame(g.ID, &PlayerVote{
		UserID:    u.ID,
		Guess:     req.Guess,
		Confirmed: req.Confirmed,
	}); err != nil {
		http.Error(w, fmt.Sprintf("failed to inform players of vote: %v", err), http.StatusInternalServerError)
		return
	}

	if !req.Confirmed {
		// If it's not confirmed (e.g. it's just tentative), so we shouldn't count
		// the votes.
		return
	}

	s.consensus.RecordVote(g.ID, u.ID, req.Guess)

	guess, hasConsensus := s.consensus.ReachedConsensus(g.ID, countVoters(prs, g.State.ActiveTeam))
	if !hasConsensus {
		return
	}

	gfm := game.NewForMove(g.State)

	newState, newStatus, err := gfm.Move(&game.Move{
		Action: game.ActionGuess,
		Team:   userPR.Team,
		Guess:  guess,
	})
	if err != nil {
		// We assume the error is the result of a bad request.
		http.Error(w, fmt.Sprintf("failed to make move : %v", err), http.StatusBadRequest)
		return
	}

	// They've made the guess, clear out the consensus for the next time.
	s.consensus.Clear(g.ID)

	// Update the state in the database.
	if err := s.db.UpdateState(g.ID, newState); err != nil {
		http.Error(w, fmt.Sprintf("failed to update game state: %v", err), http.StatusInternalServerError)
		return
	}

	var card *codenames.Card
	for _, c := range newState.Board.Cards {
		if strings.ToLower(c.Codename) == strings.ToLower(guess) {
			card = &c
			break
		}
	}

	if card == nil {
		http.Error(w, fmt.Sprintf("guess %q didn't correspond to a card", guess), http.StatusInternalServerError)
		return
	}

	if err := s.hub.ToGame(g.ID, &GuessGiven{
		Guess:        guess,
		RevealedCard: card,
	}); err != nil {
		http.Error(w, fmt.Sprintf("failed to inform players of guess: %v", err), http.StatusInternalServerError)
		return
	}

	if newStatus != codenames.Finished {
		return
	}

	over, winningTeam := gfm.GameOver()
	if !over {
		http.Error(w, "state was finished, but GameOver() says no", http.StatusInternalServerError)
		return
	}

	// The game is over, we should let folks know.
	if err := s.hub.ToGame(g.ID, &GameEnd{
		WinningTeam: winningTeam,
	}); err != nil {
		http.Error(w, fmt.Sprintf("failed to inform players of game over: %v", err), http.StatusInternalServerError)
		return
	}
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

func (s *Srv) serveData(w http.ResponseWriter, r *http.Request, u *codenames.User, game *codenames.Game, userPR *codenames.PlayerRole, prs []*codenames.PlayerRole) {
	conn, err := s.ws.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.hub.Register(conn, game.ID, u.ID)
}

type jsBoard struct {
	Cards [][]codenames.Card
}

func toJSBoard(b *codenames.Board) (*jsBoard, error) {
	sz, ok := sqrt(len(b.Cards))
	if !ok {
		return nil, fmt.Errorf("%d cards is not a square board", len(b.Cards))
	}

	cds := make([][]codenames.Card, sz)
	for i := 0; i < sz; i++ {
		cds[i] = b.Cards[i*sz : (i+1)*sz]
	}
	return &jsBoard{Cards: cds}, nil
}

func sqrt(i int) (int, bool) {
	rt := math.Floor(math.Sqrt(float64(i)))
	if int(rt*rt) != i {
		return 0, false
	}
	return int(rt), true
}

func loadKeys() (*securecookie.SecureCookie, error) {
	hashKey, err := loadOrGenKey("hashKey")
	if err != nil {
		return nil, err
	}

	blockKey, err := loadOrGenKey("blockKey")
	if err != nil {
		return nil, err
	}

	return securecookie.New(hashKey, blockKey), nil
}

func loadOrGenKey(name string) ([]byte, error) {
	f, err := ioutil.ReadFile(name)
	if err == nil {
		return f, nil
	}

	dat := securecookie.GenerateRandomKey(32)
	if dat == nil {
		return nil, errors.New("Failed to generate key")
	}

	err = ioutil.WriteFile(name, dat, 0777)
	if err != nil {
		return nil, errors.New("Error writing file")
	}
	return dat, nil
}

func jsonResp(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("jsonResp: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Srv) loadUser(r *http.Request) (*codenames.User, error) {
	c, err := r.Cookie("Authorization")
	if err == http.ErrNoCookie {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var uID codenames.UserID
	if err := s.sc.Decode("auth", c.Value, &uID); err != nil {
		// If we can't parse it, assume it's an old auth cookie and treat them as
		// not logged in.
		return nil, nil
	}

	u, err := s.db.User(uID)
	if err == codenames.ErrUserNotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return u, nil
}

type gameHandler = func(w http.ResponseWriter, r *http.Request, u *codenames.User, game *codenames.Game, userPR *codenames.PlayerRole, prs []*codenames.PlayerRole)

func (s *Srv) requireGameAuth(handler gameHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, ok := vars["id"]
		if !ok {
			http.Error(w, "no game ID provided", http.StatusBadRequest)
			return
		}
		gID := codenames.GameID(id)

		u, err := s.loadUser(r)
		if err != nil {
			log.Println("NO USER", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if u == nil {
			log.Println("NO USER")
			http.Error(w, "Not logged in", http.StatusUnauthorized)
			return
		}

		game, err := s.db.Game(gID)
		if err != nil {
			log.Println("NO GAME", err)
			http.Error(w, "failed to load game", http.StatusInternalServerError)
			return
		}

		prs, err := s.db.PlayersInGame(gID)
		if err != nil {
			log.Println("NO PRS", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		userPR, ok := findRole(u.ID, prs)
		if !ok {
			log.Println("NO U", err)
			http.Error(w, "You're not in this game", http.StatusForbidden)
			return
		}

		handler(w, r, u, game, userPR, prs)
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
