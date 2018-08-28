package web

import (
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/bcspragu/Codenames/db"
	"github.com/bcspragu/Codenames/hub"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
)

type Srv struct {
	sc  *securecookie.SecureCookie
	h   *hub.Hub
	mux *mux.Router
	db  db.DB
}

// New returns an initialized server.
func New(db db.DB) (*Srv, error) {
	sc, err := loadKeys()
	if err != nil {
		return nil, err
	}

	s := &Srv{
		sc: sc,
		h:  hub.New(),
		db: db,
	}

	s.mux = s.initMux()

	return s, nil
}

func (s *Srv) initMux() *mux.Router {
	m := mux.NewRouter()
	// New game.
	m.HandleFunc("/api/game", s.serveCreateGame).Methods("POST")
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

	// WebSocket handler for games.
	m.HandleFunc("/api/game/{id}/ws", s.serveData).Methods("GET")

	return m
}

func (s *Srv) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Srv) serveCreateGame(w http.ResponseWriter, r *http.Request) {
	return
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
