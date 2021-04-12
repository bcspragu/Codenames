package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"

	"github.com/bcspragu/Codenames/codenames"
	"github.com/bcspragu/Codenames/web"
	"github.com/olekukonko/tablewriter"
)

func main() {
	var (
		serverScheme = flag.String("server_scheme", "http", "The scheme of the server to connect to to play the game.")
		serverAddr   = flag.String("server_addr", "localhost:8080", "The address of the server to connect to to play the game.")
		gameToJoin   = flag.String("game_to_join", "", "The ID of the game to join, will create one if its blank")
		teamToJoin   = flag.String("team_to_join", "", "The name of the team to join, 'BLUE' or 'RED'")
		roleToJoin   = flag.String("role_to_join", "", "The name of the role to join, 'SPYMASTER' or 'OPERATIVE'")
	)
	flag.Parse()

	requiredFlags := []struct {
		name string
		val  *string
	}{
		{"team_to_join", teamToJoin},
		{"role_to_join", roleToJoin},
	}

	for _, rf := range requiredFlags {
		if *rf.val == "" {
			log.Fatalf("--%s must be specified", rf.name)
		}
	}

	if flag.NArg() < 1 {
		log.Fatal("need to specify a username")
	}
	name := flag.Arg(0)

	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatalf("failed to create cookie jar: %v", err)
	}

	client := &client{
		scheme: *serverScheme,
		addr:   *serverAddr,
		client: &http.Client{Jar: jar},
	}

	if err := client.createUser(name); err != nil {
		log.Fatalf("failed to create user: %v", err)
	}

	var gameID string
	if *gameToJoin == "" {
		gID, err := client.createGame()
		if err != nil {
			log.Fatalf("failed to create game: %v", err)
		}
		gameID = gID
		fmt.Printf("Created game %q\n", gameID)
	} else {
		gameID = *gameToJoin
	}

	if err := client.joinGame(gameID, *teamToJoin, *roleToJoin); err != nil {
		log.Fatalf("failed to join game: %v", err)
	}

	team, role := codenames.Team(*teamToJoin), codenames.Role(*roleToJoin)

	// Now that we've joined the game, connect via WebSockets.
	reader := bufio.NewReader(os.Stdin)

	// defer termui.Close()
	err = client.listenForUpdates(gameID, wsHooks{
		onConnect: func() {
			if *gameToJoin == "" {
				// Means we created the game, so we need to start it.
				for {
					fmt.Print("Press ENTER to start game")
					_, _ = reader.ReadString('\n')
					if err := client.startGame(gameID); err != nil {
						log.Printf("failed to start game: %v", err)
						continue
					}
					break
				}
			}
		},
		onStart: func(gs *web.GameStart) {
			// if err := termui.Init(); err != nil {
			// 	log.Fatalf("failed to initialize termui: %v", err)
			// }
			printBoard(gs.Game.State.Board)

			// If the game started, and we're the starter spymaster, give a clue.
			if role == codenames.SpymasterRole && gs.Game.State.ActiveTeam == team {
				if err := giveAClue(client, gameID, reader); err != nil {
					log.Fatalf("failed to give clue: %v", err)
				}
			}
		},
		onClueGiven: func(cg *web.ClueGiven) {
			fmt.Printf("Clue Given: %q %d\n", cg.Clue.Word, cg.Clue.Count)

			if role != codenames.OperativeRole || team != cg.Team {
				return
			}

			// If we're an operative, and the clue was given for our team, let's
			// guess.
			if err := giveAGuess(client, gameID, cg.Game.State.Board, reader); err != nil {
				log.Fatalf("failed to give clue: %v", err)
			}
		},
		onPlayerVote: func(pv *web.PlayerVote) {
			// TODO: Show the vote
		},
		onGuessGiven: func(gg *web.GuessGiven) {
			fmt.Printf("Guess was %q, card was %+v\n", gg.Guess, gg.RevealedCard)

			// We're an operative on the active team and we got the last one correct
			// and have guesses left.
			if gg.CanKeepGuessing && role == codenames.OperativeRole && team == gg.Team {
				if err := giveAGuess(client, gameID, gg.Game.State.Board, reader); err != nil {
					log.Fatalf("failed to give clue: %v", err)
				}
			}

			// We're the opposing spymaster and the other team is done guessing.
			if !gg.CanKeepGuessing && role == codenames.SpymasterRole && team != gg.Team {
				if err := giveAClue(client, gameID, reader); err != nil {
					log.Fatalf("failed to give clue: %v", err)
				}
			}
		},
		onEnd: func(ge *web.GameEnd) {
			fmt.Printf("Game over, %q won!", ge.WinningTeam)
		},
	})
	if err != nil {
		log.Fatalf("failed to listen for updates: %v", err)
	}
}

func giveAClue(client *client, gameID string, reader *bufio.Reader) error {
	clue := getAClue(reader)
	if err := client.giveClue(gameID, clue); err != nil {
		return fmt.Errorf("failed to send clue: %w", err)
	}
	return nil
}

func getAClue(reader *bufio.Reader) *codenames.Clue {
	for {
		fmt.Print("Enter a clue: ")
		clueStr, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("failed to read clue: %v", err)
		}
		clue, err := codenames.ParseClue(strings.TrimSpace(clueStr))
		if err != nil {
			fmt.Printf("malformed clue, please try again: %v", err)
			continue
		}
		return clue
	}
}

func giveAGuess(client *client, gameID string, board *codenames.Board, reader *bufio.Reader) error {
	guess, confirmed := getAGuess(reader, board)
	if err := client.giveGuess(gameID, guess, confirmed); err != nil {
		return fmt.Errorf("failed to send guess: %w", err)
	}
	return nil
}

func getAGuess(reader *bufio.Reader, board *codenames.Board) (string, bool) {
	for {
		fmt.Print("Enter a guess: ")
		guess, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("failed to read guess: %v", err)
		}
		guess = strings.ToLower(strings.TrimSpace(guess))
		if !guessInCards(guess, board.Cards) {
			fmt.Println("guess was not found on board, please try again")
			continue
		}

		fmt.Print("Confirmed? (Y/n): ")
		confirmedStr, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("failed to read guess: %v", err)
		}
		confirmedStr = strings.TrimSpace(confirmedStr)
		confirmed := len(confirmedStr) == 0 || strings.ToLower(confirmedStr[0:1]) == "y"

		return guess, confirmed
	}
}

func guessInCards(guess string, cards []codenames.Card) bool {
	for _, c := range cards {
		if guess == strings.ToLower(c.Codename) {
			return true
		}
	}
	return false
}

type client struct {
	scheme string
	addr   string
	client *http.Client
}

func (c *client) createUser(name string) error {
	body := struct {
		Name string `json:"name"`
	}{name}

	req, err := http.NewRequest(http.MethodPost, c.scheme+"://"+c.addr+"/api/user", toBody(body))
	if err != nil {
		return fmt.Errorf("failed to form request: %w", err)
	}

	if err := c.do(req, nil); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (c *client) createGame() (string, error) {
	req, err := http.NewRequest(http.MethodPost, c.scheme+"://"+c.addr+"/api/game", nil)
	if err != nil {
		return "", fmt.Errorf("failed to form request: %w", err)
	}

	var resp struct {
		ID string `json:"id"`
	}
	if err := c.do(req, &resp); err != nil {
		return "", fmt.Errorf("failed to create game: %w", err)
	}
	return resp.ID, nil
}

func (c *client) joinGame(gID, team, role string) error {
	body := struct {
		Team string `json:"team"`
		Role string `json:"role"`
	}{team, role}

	req, err := http.NewRequest(http.MethodPost, c.scheme+"://"+c.addr+"/api/game/"+gID+"/join", toBody(body))
	if err != nil {
		return fmt.Errorf("failed to form request: %w", err)
	}

	if err := c.do(req, nil); err != nil {
		return fmt.Errorf("failed to join game: %w", err)
	}

	return nil
}

func (c *client) startGame(gID string) error {
	req, err := http.NewRequest(http.MethodPost, c.scheme+"://"+c.addr+"/api/game/"+gID+"/start", nil)
	if err != nil {
		return fmt.Errorf("failed to form request: %w", err)
	}

	if err := c.do(req, nil); err != nil {
		return fmt.Errorf("failed to start game: %w", err)
	}

	return nil
}

func (c *client) giveClue(gID string, clue *codenames.Clue) error {
	body := struct {
		Word  string `json:"word"`
		Count int    `json:"count"`
	}{clue.Word, clue.Count}

	req, err := http.NewRequest(http.MethodPost, c.scheme+"://"+c.addr+"/api/game/"+gID+"/clue", toBody(body))
	if err != nil {
		return fmt.Errorf("failed to form request: %w", err)
	}

	if err := c.do(req, nil); err != nil {
		return fmt.Errorf("failed to give clue to game: %w", err)
	}

	return nil
}

func (c *client) giveGuess(gID, guess string, confirmed bool) error {
	body := struct {
		Guess     string `json:"guess"`
		Confirmed bool   `json:"confirmed"`
	}{guess, confirmed}

	req, err := http.NewRequest(http.MethodPost, c.scheme+"://"+c.addr+"/api/game/"+gID+"/guess", toBody(body))
	if err != nil {
		return fmt.Errorf("failed to form request: %w", err)
	}

	if err := c.do(req, nil); err != nil {
		return fmt.Errorf("failed to give clue to game: %w", err)
	}

	return nil
}

func (c *client) do(req *http.Request, resp interface{}) error {
	httpResp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusOK {
		return handleError(httpResp)
	}

	if resp != nil {
		if err := json.NewDecoder(httpResp.Body).Decode(resp); err != nil {
			return fmt.Errorf("failed to decode response body: %w", err)
		}
	}

	return nil
}

type httpError struct {
	statusCode int
	body       string
	err        error
}

func (h *httpError) Error() string {
	if h.err != nil {
		return fmt.Sprintf("[%d] failed to handle error: %v", h.statusCode, h.err)
	}
	return fmt.Sprintf("[%d] error from server: %s", h.statusCode, h.body)
}

func handleError(resp *http.Response) error {
	dat, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return &httpError{
			statusCode: resp.StatusCode,
			err:        fmt.Errorf("failed to read error response body: %w", err),
		}
	}

	return &httpError{
		statusCode: resp.StatusCode,
		body:       string(dat),
	}
}

func toBody(req interface{}) io.Reader {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(req); err != nil {
		return &errReader{err: err}
	}
	return &buf
}

type errReader struct {
	err error
}

func (e *errReader) Read(_ []byte) (int, error) {
	return 0, e.err
}

func printBoard(b *codenames.Board) {
	table := tablewriter.NewWriter(os.Stdout)

	for i := 0; i < 5; i++ {
		var row []string
		var colors []tablewriter.Colors
		for j := 0; j < 5; j++ {
			card := b.Cards[i*5+j]
			var c tablewriter.Colors
			switch card.Agent {
			case codenames.BlueAgent:
				c = append(c, tablewriter.FgBlueColor)
			case codenames.RedAgent:
				c = append(c, tablewriter.FgHiRedColor)
			case codenames.Assassin:
				c = append(c, tablewriter.BgHiRedColor)
			}
			if card.Revealed {
				c = append(c, tablewriter.UnderlineSingle)
			}
			colors = append(colors, c)
			row = append(row, card.Codename)
		}
		table.Rich(row, colors)
	}

	table.Render()
}
