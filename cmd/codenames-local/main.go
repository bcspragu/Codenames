package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	codenames "github.com/bcspragu/Codenames"
	"github.com/bcspragu/Codenames/game"
	"github.com/bcspragu/Codenames/io"
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

	words := strings.Split(*wordList, ",")
	if len(words) != codenames.Size {
		log.Fatalf("Expected %d words, got %d words", codenames.Size, len(words))
	}

	// Initialize our word2vec model.
	ai, err := w2v.New(*modelFile)
	if err != nil {
		log.Fatalf("Failed to initialize word2vec model: %v", err)
	}

	var (
		rsm codenames.Spymaster = &io.Spymaster{In: os.Stdin, Out: os.Stdout, Team: codenames.RedTeam}
		bsm codenames.Spymaster = &io.Spymaster{In: os.Stdin, Out: os.Stdout, Team: codenames.BlueTeam}
		rop codenames.Operative = &io.Operative{In: os.Stdin, Out: os.Stdout, Team: codenames.RedTeam}
		bop codenames.Operative = &io.Operative{In: os.Stdin, Out: os.Stdout, Team: codenames.BlueTeam}
	)

	switch teamMap[*starter] {
	case codenames.RedTeam:
		rop = ai
	case codenames.BlueTeam:
		bop = ai
	}

	cards := make([]codenames.Card, len(words))
	agents := []codenames.Agent{
		codenames.RedAgent,
		codenames.RedAgent,
		codenames.RedAgent,
		codenames.RedAgent,
		codenames.RedAgent,
		codenames.RedAgent,
		codenames.RedAgent,
		codenames.RedAgent,
		codenames.RedAgent,
		codenames.BlueAgent,
		codenames.BlueAgent,
		codenames.BlueAgent,
		codenames.BlueAgent,
		codenames.BlueAgent,
		codenames.BlueAgent,
		codenames.BlueAgent,
		codenames.BlueAgent,
		codenames.Bystander,
		codenames.Bystander,
		codenames.Bystander,
		codenames.Bystander,
		codenames.Bystander,
		codenames.Bystander,
		codenames.Bystander,
		codenames.Assassin,
	}
	for i, w := range words {
		cards[i].Codename = w
		cards[i].Agent = agents[i]
	}
	b := &codenames.Board{Cards: cards}
	g, err := game.New(b, &game.Config{
		Starter:       teamMap[*starter],
		RedSpymaster:  rsm,
		BlueSpymaster: bsm,
		RedOperative:  rop,
		BlueOperative: bop,
	})
	if err != nil {
		log.Fatalf("Failed to instantiate game: %v", err)
	}

	fmt.Println(g.Play())
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
