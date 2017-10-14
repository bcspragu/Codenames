package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/bcspragu/Codenames/codenames"
	"github.com/bcspragu/Codenames/dict"
	"github.com/bcspragu/Codenames/types"
	"github.com/bcspragu/Codenames/w2v"
	"google.golang.org/api/option"
)

type ioGame struct {
	sc   *bufio.Scanner
	game *codenames.Game
}

var (
	me   = types.RedTeam
	them = types.BlueTeam
)

func main() {
	var (
		//addr       = flag.String("addr", ":8080", "The port to run the web server on.")
		apiKeyFile = flag.String("api_key_file", "", "The file containing an API key to authenticate with Google Cloud")
		dictFile   = flag.String("dict_file", "words.txt", "A newline-separated dictionary of words, all in uppercase letters.")
		modelFile  = flag.String("model_file", "w2v.bin", "A binary-formatted word2vec pre-trained model file.")
		wordList   = flag.String("words", "", "Comma-separated list of words.")
		first      = flag.Bool("first", true, "Whether or not this player goes first.")
		team       = flag.String("team", "red", "Team to be")
	)
	flag.Parse()

	if *team == "blue" {
		them = types.RedTeam
		me = types.BlueTeam
	}

	//ctx := context.Background()
	var opts []option.ClientOption
	// If an API key is specified, clear the GOOGLE_APPLICATION_CREDENTIALS and
	// use the key instead.
	if *apiKeyFile != "" {
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")
		d, err := ioutil.ReadFile(*apiKeyFile)
		if err != nil {
			log.Fatalf("Failed to read API key: %v", err)
		}

		apiKey := string(d)
		opts = append(opts, option.WithAPIKey(apiKey))
	}

	//cvtr, err := vision.New(ctx, opts...)
	//if err != nil {
	//log.Fatalf("Failed to instantiate converter: %v", err)
	//}

	// Initialize our dictionary.
	if err := dict.Init(*dictFile); err != nil {
		log.Fatalf("Failed to initialize dictionary: %v", err)
	}

	// Initialize our word2vec model.
	if err := w2v.Init(*modelFile); err != nil {
		log.Fatalf("Failed to initialize word2vec model: %v", err)
	}
	//srv, err := server.New(cvtr)
	//if err != nil {
	//log.Fatalf("Failed to initialize server: %v", err)
	//}
	//http.ListenAndServe(*addr, srv)

	words := strings.Split(*wordList, ",")
	if len(words) != types.Size {
		log.Fatalf("Expected %d words, got %d words", types.Size, len(words))
	}

	cns := make([]types.Codename, len(words))
	for i, w := range words {
		cns[i].Name = w
	}

	b := &types.Board{Codenames: cns}
	g, err := codenames.NewGame(b, &codenames.Config{
		Team: me,
		Role: types.Operative,
	})
	if err != nil {
		log.Fatalf("Failed to instantiate game: %v", err)
	}

	sc := bufio.NewScanner(os.Stdin)
	iog := &ioGame{
		sc:   sc,
		game: g,
	}
	iog.gameLoop(*first)
}

func (i *ioGame) gameLoop(first bool) {
	for i.sc.Scan() {
		if first {
			i.enterClue()
			i.otherTeam()
		} else {
			i.otherTeam()
			i.enterClue()
		}
	}
}

func (i *ioGame) otherTeam() {
	fmt.Println("Did other team get anything?")
	i.sc.Scan()
	txt := i.sc.Text()
	if txt == "" {
		return
	}
	words := strings.Split(txt, " ")
	for _, word := range words {
		i.game.Assign(word, them)
	}
}

func (i *ioGame) enterClue() {
	fmt.Println("Enter clue: ")
	i.sc.Scan()
	guesses, err := i.game.Guess(i.sc.Text())
	if err != nil {
		log.Printf("Failed to guess: %v", err)
	}
	for n, g := range guesses {
		fmt.Printf("Guess #%d = %q, correct? ", n, g)
		i.sc.Scan()
		if i.sc.Text() != "y" {
			break
		}
		i.game.Assign(g, me)
	}
}
