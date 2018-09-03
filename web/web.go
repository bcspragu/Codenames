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

	codenames "github.com/bcspragu/Codenames"
	"github.com/bcspragu/Codenames/boardgen"
	"github.com/bcspragu/Codenames/hub"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
)

type Srv struct {
	sc  *securecookie.SecureCookie
	h   *hub.Hub
	mux *mux.Router
	db  codenames.DB
	r   *rand.Rand
}

// New returns an initialized server.
func New(db codenames.DB, r *rand.Rand) (*Srv, error) {
	sc, err := loadKeys()
	if err != nil {
		return nil, err
	}

	s := &Srv{
		sc: sc,
		h:  hub.New(),
		db: db,
		r:  r,
	}

	s.mux = s.initMux()

	return s, nil
}

func (s *Srv) initMux() *mux.Router {
	m := mux.NewRouter()
	// New game.
	m.HandleFunc("/api/game", s.serveCreateGame).Methods("POST")
	// Pending games.
	m.HandleFunc("/api/games", s.servePendingGames).Methods("GET")
	// Get game.
	m.HandleFunc("/api/game/{id}", s.serveGame).Methods("GET")
	// Join game.
	m.HandleFunc("/api/game/{id}/join", s.serveJoinGame).Methods("POST")
	// Start game.
	m.HandleFunc("/api/game/{id}/start", s.serveStartGame).Methods("POST")
	// Serve a clue to a game.
	m.HandleFunc("/api/game/{id}/clue", s.serveClue).Methods("POST")
	// Serve a card guess to a game.
	m.HandleFunc("/api/game/{id}/guess", s.serveGuess).Methods("POST")

	// TODO(bcspragu): Remove this handler, it's just for testing HTTP requests.
	m.HandleFunc("/api/newBoard", s.serveBoard).Methods("GET")

	// WebSocket handler for games.
	m.HandleFunc("/api/game/{id}/ws", s.serveData).Methods("GET")

	return m
}

func (s *Srv) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Srv) serveCreateGame(w http.ResponseWriter, r *http.Request) {
	ar := codenames.RedTeam
	if s.r.Intn(2) == 0 {
		ar = codenames.BlueTeam
	}

	id, err := s.db.NewGame(&codenames.Game{
		State: &codenames.GameState{
			ActiveTeam: ar,
			ActiveRole: codenames.SpymasterRole,
			Board:      boardgen.New(ar, s.r),
		},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(struct{ ID string }{string(id)}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Srv) servePendingGames(w http.ResponseWriter, r *http.Request) {
	gIDs, err := s.db.PendingGames()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(gIDs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Srv) serveGame(w http.ResponseWriter, r *http.Request) {

}

func (s *Srv) serveJoinGame(w http.ResponseWriter, r *http.Request) {

}

func (s *Srv) serveStartGame(w http.ResponseWriter, r *http.Request) {

}

func (s *Srv) serveClue(w http.ResponseWriter, r *http.Request) {

}

func (s *Srv) serveGuess(w http.ResponseWriter, r *http.Request) {

}

func (s *Srv) serveData(w http.ResponseWriter, r *http.Request) {

}

type jsBoard struct {
	Cards [][]codenames.Card
}

func (s *Srv) serveBoard(w http.ResponseWriter, r *http.Request) {
	b, err := toJSBoard(boardgen.New(codenames.RedTeam, s.r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(b); err != nil {
		log.Printf("serveBoard: %v", err)
	}
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
