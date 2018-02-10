package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	codenames "github.com/bcspragu/Codenames"
	"github.com/bcspragu/Codenames/game"
	"github.com/bcspragu/Codenames/w2v"
)

var (
	teamMap = map[string]codenames.Team{
		"red":  codenames.RedTeam,
		"blue": codenames.BlueTeam,
	}
)

func main() {
	var (
		modelFile = flag.String("model_file", "w2v.bin", "A binary-formatted word2vec pre-trained model file.")
		wordList  = flag.String("words", "", "Comma-separated list of words.")
		starter   = flag.String("starter", "red", "Which color team starts the game")
		team      = flag.String("team", "red", "Team to be")
	)
	flag.Parse()

	if err := validColor(*starter); err != nil {
		log.Fatal(err)
	}

	if err := validColor(*team); err != nil {
		log.Fatal(err)
	}

	// Initialize our word2vec model.
	ai, err := w2v.New(*modelFile)
	if err != nil {
		log.Fatalf("Failed to initialize word2vec model: %v", err)
	}

	words := strings.Split(*wordList, ",")
	if len(words) != codenames.Size {
		log.Fatalf("Expected %d words, got %d words", codenames.Size, len(words))
	}

	cns := make([]codenames.Card, len(words))
	for i, w := range words {
		cns[i].Codename = w
	}

	b := &codenames.Board{}
	g, err := game.New(b, &game.Config{
		Team:        teamMap[*team],
		Starter:     teamMap[*starter],
		Role:        codenames.Operative,
		OperativeAI: ai,
	})
	if err != nil {
		log.Fatalf("Failed to instantiate game: %v", err)
	}
	// TODO: Figure out how the game object should be played.
	_ = g
}

func validColor(c string) error {
	switch c {
	case "red":
		return nil
	case "blue":
		return nil
	default:
		return fmt.Errorf("invalid team color %q, 'red' and 'blue' are the only valid team colors", c)
	}
}
