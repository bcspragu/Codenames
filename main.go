package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/bcspragu/Codenames/codenames"
	"github.com/bcspragu/Codenames/types"
	"github.com/bcspragu/Codenames/vision"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
)

func main() {
	ctx := context.Background()
	d, err := ioutil.ReadFile("apiKey")
	if err != nil {
		log.Fatalf("Failed to read API key: %v", err)
	}

	apiKey := string(d)
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "")

	cvtr, err := vision.New(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Fatalf("Failed to instantiate converter: %v", err)
	}

	f, err := os.Open("codenames.jpg")
	if err != nil {
		log.Fatalf("Failed to open new image: %v", err)
	}
	defer f.Close()

	b, err := cvtr.BoardFromReader(ctx, f)
	if err != nil {
		log.Fatalf("Failed to generate board: %v", err)
	}

	g, err := codenames.NewGame(b, &codenames.Config{
		Team: types.RedTeam,
		Role: types.Operative,
	})
	if err != nil {
		log.Fatalf("Failed to instantiate game: %v", err)
	}

	sc := bufio.NewScanner(os.Stdin)
	fmt.Println("Game started, enter clue: ")
	for sc.Scan() {
		guesses, err := g.Guess(sc.Text())
		if err != nil {
			log.Printf("Failed to guess: %v", err)
		}
		for i, g := range guesses {
			fmt.Printf("Guess #%d = %q\n", i, g)
		}
		fmt.Println("Board loaded, enter clue: ")
	}
}
