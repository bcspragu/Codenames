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
	"time"

	"github.com/bcspragu/Codenames/codenames"
	"github.com/bcspragu/Codenames/web"
	"github.com/gorilla/websocket"
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

	if len(os.Args) < 3 {
		log.Fatal("need to specify a username and a mode")
	}
	name := os.Args[1]

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
	wsClient, err := client.listenForUpdates(gameID)
	if err != nil {
		log.Fatalf("failed to listen for update: %v", err)
	}

	reader := bufio.NewReader(os.Stdin)
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

	gameStart := wsClient.waitForGameToStart()

	yourMove := role == codenames.SpymasterRole && gameStart.Game.State.ActiveTeam == team

	for {
		if yourMove {
			switch role {
			case codenames.SpymasterRole:
				if err := giveAClue(client, gameID, reader); err != nil {
					log.Fatalf("failed to give clue: %v", err)
				}
			case codenames.OperativeRole:
				if err := giveAGuess(client, gameID, gameStart.Game.State.Board, reader); err != nil {
					log.Fatalf("failed to give clue: %v", err)
				}
			}
		}

		wsClient.waitForTurn(team, role)
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
		clue, err := codenames.ParseClue(clueStr)
		if err != nil {
			fmt.Println("malformed clue, please try again")
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
		guess = strings.ToLower(guess)
		if !guessInCards(guess, board.Cards) {
			fmt.Println("guess was not found on board, please try again")
			continue
		}

		fmt.Print("Confirmed? (Y/n): ")
		confirmedStr, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("failed to read guess: %v", err)
		}
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

func (c *client) listenForUpdates(gID string) (*wsClient, error) {
	scheme := "ws"
	if c.scheme == "https" {
		scheme = "wss"
	}

	addr := scheme + "://" + c.addr + "/api/game/" + gID + "/ws"

	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
		Jar:              c.client.Jar,
	}
	conn, _, err := dialer.Dial(addr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %w", err)
	}

	wsc := &wsClient{
		conn:      conn,
		done:      make(chan struct{}),
		gameStart: make(chan *web.GameStart, 1),
	}
	go wsc.read()

	return wsc, nil
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
