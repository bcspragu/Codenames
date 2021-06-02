package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"

	"code.sajari.com/word2vec"
	"github.com/bcspragu/Codenames/cryptorand"
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

	model, err := loadModel(*modelPath)
	if err != nil {
		log.Fatalf("failed to load model: %v", err)
	}

	r := rand.New(cryptorand.NewSource())

	srv := newServer(model, *authSecret, *webServerScheme, *webServerAddr, r)

	if err := http.ListenAndServe(":8080", srv); err != nil {
		log.Fatalf("error from server: %v", err)
	}
}

func loadModel(path string) (*word2vec.Model, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read model path: %w", err)
	}
	defer f.Close()

	model, err := word2vec.FromReader(f)
	if err != nil {
		return nil, fmt.Errorf("failed to load model: %w", err)
	}

	return model, nil
}
