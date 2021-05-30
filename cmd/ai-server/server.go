package main

import (
	"log"
	"net/http"

	"code.sajari.com/word2vec"
	"github.com/bcspragu/Codenames/httperr"
)

type Server struct {
	model      *word2vec.Model
	mux        *http.ServeMux
	authSecret string
}

func newServer(model *word2vec.Model, authSecret string) *Server {
	srv := &Server{
		model:      model,
		authSecret: authSecret,
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

	// TODO(bcspragu): Connect to the server using the client, join the game,
	// etc, etc.
	return nil
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
