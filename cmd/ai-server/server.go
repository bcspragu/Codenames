package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/bcspragu/Codenames/client"
	"github.com/bcspragu/Codenames/codenames"
	"github.com/bcspragu/Codenames/httperr"
	"github.com/bcspragu/Codenames/w2v"
	"github.com/bcspragu/Codenames/web"
)

const (
	maxConcurrentGames = 25
)

type activePlayer struct {
	gameID codenames.GameID
}

type Server struct {
	ai              *w2v.AI
	authSecret      string
	webServerScheme string
	webServerAddr   string
	r               *rand.Rand

	mux *http.ServeMux

	mu            sync.Mutex
	activePlayers map[codenames.RobotID]*activePlayer
}

func newServer(ai *w2v.AI, authSecret, webServerScheme, webServerAddr string, r *rand.Rand) *Server {
	srv := &Server{
		ai:              ai,
		authSecret:      authSecret,
		webServerScheme: webServerScheme,
		webServerAddr:   webServerAddr,
		r:               r,
		activePlayers:   make(map[codenames.RobotID]*activePlayer),
	}
	srv.initMux()
	return srv
}

func (s *Server) initMux() {
	mux := http.NewServeMux()
	mux.HandleFunc("/join", s.handleError(s.serveJoin))
	s.mux = mux
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) serveJoin(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		return httperr.MethodNotAllowed("call to join with bad method %q", r.Method)
	}
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return httperr.
			Unauthorized("no auth in join request").
			WithMessage("no auth given")
	}
	if auth != s.authSecret {
		return httperr.
			Forbidden("bad auth secret in join requesrt").
			WithMessage("invalid auth")
	}

	var req struct {
		GameID string `json:"game_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return httperr.BadRequest("failed to decode join request: %w", err)
	}

	if req.GameID == "" {
		return httperr.
			BadRequest("no game ID given").
			WithMessage("no game ID given")
	}
	gID := codenames.GameID(req.GameID)

	name := s.aiName()

	c, err := client.New(s.webServerScheme, s.webServerAddr)
	if err != nil {
		return httperr.Internal("failed to init Codenames client: %w", err)
	}

	pID, err := c.CreateUser(name, codenames.PlayerTypeRobot)
	if err != nil {
		return httperr.Internal("failed to create user %q: %w", name, err)
	}
	rID := codenames.RobotID(pID)

	if err := c.JoinGame(gID); err != nil {
		return httperr.Internal("failed to join game %q: %w", gID, err)
	}

	s.mu.Lock()
	if len(s.activePlayers) >= maxConcurrentGames {
		s.mu.Unlock()
		return httperr.
			Teapot("can't join a game when we already have %d active games", len(s.activePlayers)).
			WithMessage("too many active AIs, try again later")
	}
	s.activePlayers[rID] = &activePlayer{gameID: gID}
	s.mu.Unlock()
	log.Printf("Created player %q in game %q", rID, gID)

	// Background the process of playingk
	go func() {
		defer s.unlockPlayer(rID)

		s.playGame(c, gID, rID)
	}()

	return jsonResp(w, struct {
		RobotID string `json:"robot_id"`
		Success bool   `json:"success"`
	}{string(rID), true})
}

func (s *Server) unlockPlayer(rID codenames.RobotID) {
	s.mu.Lock()
	delete(s.activePlayers, rID)
	s.mu.Unlock()
}

func (s *Server) playGame(c *client.Client, gID codenames.GameID, rID codenames.RobotID) {
	var (
		role     codenames.Role
		team     codenames.Team
		lastClue *codenames.Clue
	)

	err := c.ListenForUpdates(gID, client.WSHooks{
		OnConnect: func() {
			// TODO(bcspragu): Decide if we need to do anything once we connect.
		},
		OnStart: func(gs *web.GameStart) {
			for _, p := range gs.Players {
				if !p.PlayerID.IsRobot(rID) {
					continue
				}
				role = p.Role
				team = p.Team
				break
			}

			if role == codenames.SpymasterRole && gs.Game.State.ActiveTeam == team {
				clue, err := s.giveClue(gs.Game.State.Board, toAgent(team))
				if err != nil {
					log.Printf("[ERROR] failed to make a clue: %v", err)
					return
				}

				if err := c.GiveClue(gID, clue); err != nil {
					log.Printf("[ERROR] failed to give clue: %v", err)
					return
				}
			}
		},
		OnClueGiven: func(cg *web.ClueGiven) {
			if cg.Team == team {
				lastClue = cg.Clue
			}

			if role != codenames.OperativeRole || cg.Team != team {
				fmt.Printf("Clue was given, but I'm a/an %q on team %q\n", role, team)
				return
			}
			fmt.Println("Clue was given, and I'm guessing!")

			guess, err := s.guess(cg.Game.State.Board, cg.Clue)
			if err != nil {
				log.Printf("[ERROR] failed to make a guess for clue %+v: %v", cg.Clue, err)
				return
			}

			if err := c.GiveGuess(gID, guess, true /* confirmed */); err != nil {
				log.Printf("[ERROR] failed to give guess %q for clue %+v: %v", guess, cg.Clue, err)
				return
			}
		},
		OnGuessGiven: func(gg *web.GuessGiven) {
			// We only want to formulate a clue when the *other* team has just
			// finished guessing.
			if gg.Team != team && !gg.CanKeepGuessing && role == codenames.SpymasterRole {
				fmt.Println("My turn to clue!")

				clue, err := s.giveClue(gg.Game.State.Board, toAgent(team))
				if err != nil {
					log.Printf("[ERROR] failed to make a clue: %v", err)
					return
				}

				if err := c.GiveClue(gID, clue); err != nil {
					log.Printf("[ERROR] failed to give clue: %v", err)
					return
				}

				return
			}

			if gg.Team == team && gg.CanKeepGuessing && role == codenames.OperativeRole {
				fmt.Println("I can keep guessing!")

				guess, err := s.guess(gg.Game.State.Board, lastClue)
				if err != nil {
					log.Printf("[ERROR] failed to make a guess for clue %+v: %v", lastClue, err)
					return
				}

				if err := c.GiveGuess(gID, guess, true /* confirmed */); err != nil {
					log.Printf("[ERROR] failed to give guess %q for clue %+v: %v", guess, lastClue, err)
					return
				}

				return
			}
		},
	})
	if err != nil {
		log.Printf("[ERROR] error listening for updates in game %q: %v", gID, err)
	}
}

func (s *Server) giveClue(b *codenames.Board, agent codenames.Agent) (*codenames.Clue, error) {
	clue, err := s.ai.GiveClue(b, agent)
	if err != nil {
		log.Printf("[ERROR] AI failed to make a clue: %v", err)
		return &codenames.Clue{
			Word:  "lolgoodluck",
			Count: 1,
		}, nil
	}

	return clue, nil
}

func toAgent(team codenames.Team) codenames.Agent {
	switch team {
	case codenames.BlueTeam:
		return codenames.BlueAgent
	case codenames.RedTeam:
		return codenames.RedAgent
	default:
		return codenames.UnknownAgent
	}
}

func (s *Server) guess(b *codenames.Board, clue *codenames.Clue) (string, error) {
	guess, err := s.ai.Guess(b, clue)
	if err != nil {
		log.Printf("[ERROR] AI failed to make a guess: %v", err)
		return s.guessRandomly(b, clue)
	}
	return guess, nil
}

func (s *Server) guessRandomly(b *codenames.Board, clue *codenames.Clue) (string, error) {
	unused := codenames.Unused(b.Cards)
	if len(unused) == 0 {
		return "", errors.New("no available cards left on the board")
	}

	return unused[s.r.Intn(len(unused))].Codename, nil
}

func (s *Server) aiName() string {
	var buf strings.Builder
	buf.WriteString("AI-")
	buf.WriteString(strconv.Itoa(s.r.Int()))
	return buf.String()
}

type handlerFunc func(w http.ResponseWriter, r *http.Request) error

func (s *Server) handleError(h handlerFunc) http.HandlerFunc {
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

func jsonResp(w http.ResponseWriter, v interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return httperr.Internal("failed to encode response for %+v of type %T: %w", v, v, err)
	}

	return nil
}
