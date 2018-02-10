// codenames-local runs a game of
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
	agentMap = map[string]codenames.Agent{
		"red":       codenames.RedAgent,
		"blue":      codenames.BlueAgent,
		"bystander": codenames.Bystander,
		"assassin":  codenames.Assassin,
	}
)

func main() {
	var (
		modelFile = flag.String("model_file", "w2v.bin", "A binary-formatted word2vec pre-trained model file.")
		wordList  = flag.String("words", "", "Comma-separated list of words and the agent they're assigned to. Ex dog:red,wallet:blue,bowl:assassin,glass:blue,hood:bystander")
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
	for i, w := range words {
		c, err := parseCard(w)
		if err != nil {
			log.Fatalf("Failed on card #%d: %q: %v", i, w, err)
		}
		cards[i] = c
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

func parseCard(in string) (codenames.Card, error) {
	ps := strings.Split(clue, ":")
	if len(ps) != 2 {
		return codenames.Card{}, fmt.Errorf("malformed card string %q", in)
	}

	ag, ok := agentMap[strings.ToLower(ps[1])]
	if !ok {
		return codenames.Card{}, fmt.Errorf("invalid agent type %q", ps[1])
	}

	return codenames.Card{Codename: ps[0], Agent: ag}, nil
}
