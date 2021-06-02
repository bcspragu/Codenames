package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"

	"code.sajari.com/word2vec"
	"github.com/bcspragu/Codenames/client"
	"github.com/bcspragu/Codenames/codenames"
	"github.com/bcspragu/Codenames/httperr"
	"github.com/bcspragu/Codenames/web"
)

type Server struct {
	model           *word2vec.Model
	authSecret      string
	webServerScheme string
	webServerAddr   string
	r               *rand.Rand

	mux *http.ServeMux
}

func newServer(model *word2vec.Model, authSecret, webServerScheme, webServerAddr string, r *rand.Rand) *Server {
	srv := &Server{
		model:           model,
		authSecret:      authSecret,
		webServerScheme: webServerScheme,
		webServerAddr:   webServerAddr,
		r:               r,
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

	uID, err := c.CreateUser(name)
	if err != nil {
		return httperr.Internal("failed to create user %q: %w", name, err)
	}
	rID := codenames.RobotID(uID)

	if err := c.JoinGame(gID, codenames.PlayerTypeRobot); err != nil {
		return httperr.Internal("failed to join game %q: %w", gID, err)
	}

	var (
		role codenames.Role
		team codenames.Team
	)

	err = c.ListenForUpdates(gID, client.WSHooks{
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
				return
			}
		},
		OnClueGiven: func(cg *web.ClueGiven) {
			if role != codenames.OperativeRole || cg.Team != team {
				return
			}

			// TODO(bcspragu): If we're here, come up with our answer to the clue.
		},
		OnGuessGiven: func(gg *web.GuessGiven) {
			// We only want to formulate a clue when the *other* team has just
			// finished guessing.
			if role != codenames.SpymasterRole || gg.Team == team || gg.CanKeepGuessing {
				return
			}

			// TODO(bcspragu): If we're here, come up with our clue.
		},
	})
	if err != nil {
		return httperr.Internal("error listening for updates in game %q: %w", gID, err)
	}

	return nil
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
