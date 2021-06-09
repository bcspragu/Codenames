package main

import (
	"log"
	"math/rand"
	"net/http"

	"github.com/bcspragu/Codenames/cryptorand"
	"github.com/bcspragu/Codenames/w2v"
	"github.com/namsral/flag"
)

func main() {
	var (
		modelPath       = flag.String("model_path", "", "Path to binary model data")
		authSecret      = flag.String("auth_secret", "", "Secret string that callers must provide")
		webServerScheme = flag.String("web_server_scheme", "", "The protocol to connect to the Codenames game web server")
		webServerAddr   = flag.String("web_server_addr", "", "The address to connect to the Codenames game web server")
	)
	flag.Parse()

	if *modelPath == "" {
		log.Fatal("--model_path must be provided")
	}

	if *authSecret == "" {
		log.Fatal("--auth_secret must be provided")
	}

	if *webServerScheme == "" {
		log.Fatal("--web_server_scheme must be provided")
	}

	if *webServerAddr == "" {
		log.Fatal("--web_server_addr must be provided")
	}

	ai, err := w2v.New(*modelPath)
	if err != nil {
		log.Fatalf("failed to load AI: %v", err)
	}

	r := rand.New(cryptorand.NewSource())

	srv := newServer(ai, *authSecret, *webServerScheme, *webServerAddr, r)

	if err := http.ListenAndServe(":8081", srv); err != nil {
		log.Fatalf("error from server: %v", err)
	}
}
