package server

import (
	"crypto/rand"
	"fmt"
	"html/template"
	"net/http"
	"sync"

	"github.com/bcspragu/Codenames/codenames"
	"github.com/bcspragu/Codenames/vision"
)

type uuid string

type Server struct {
	mu    sync.Mutex
	games map[uuid]*codenames.Game

	// For parsing boards from images
	cvtr *vision.Converter

	mux  *http.ServeMux
	tmpl *template.Template
}

func New(cvtr *vision.Converter) (*Server, error) {
	tmpl, err := template.ParseGlob("templates/*.html")
	if err != nil {
		return nil, err
	}
	s := &Server{
		games: make(map[uuid]*codenames.Game),
		cvtr:  cvtr,
		mux:   http.NewServeMux(),
		tmpl:  tmpl,
	}
	s.mux.HandleFunc("/", s.indexHandler)
	s.mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("assets"))))
	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) {
	if err := s.tmpl.ExecuteTemplate(w, "index.html", nil); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
		return
	}
}

func newUUID() uuid {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		fmt.Println("Error: ", err)
		return uuid("error")
	}
	return uuid(fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]))
}
