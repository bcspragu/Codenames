package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/bcspragu/Codenames/dict"
	"github.com/bcspragu/Codenames/server"
	"github.com/bcspragu/Codenames/vision"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
)

func main() {
	var (
		addr       = flag.String("addr", ":8080", "The port to run the web server on.")
		apiKeyFile = flag.String("api_key_file", "", "The file containing an API key to authenticate with Google Cloud")
		dictFile   = flag.String("dict_file", "words.txt", "A newline-separated dictionary of words, all in uppercase letters.")
		//modelFile  = flag.String("model_file", "w2v.bin", "A binary-formatted word2vec pre-trained model file.")
	)
	flag.Parse()

	ctx := context.Background()
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

	cvtr, err := vision.New(ctx, opts...)
	if err != nil {
		log.Fatalf("Failed to instantiate converter: %v", err)
	}

	// Initialize our dictionary.
	if err := dict.Init(*dictFile); err != nil {
		log.Fatalf("Failed to initialize dictionary: %v", err)
	}

	// Initialize our word2vec model.
	//if err := w2v.Init(*modelFile); err != nil {
	//log.Fatalf("Failed to initialize word2vec model: %v", err)
	//}
	srv, err := server.New(cvtr)
	if err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}
	http.ListenAndServe(*addr, srv)

	/*
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
	*/
}
