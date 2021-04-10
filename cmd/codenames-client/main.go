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
	"time"

	"github.com/bcspragu/Codenames/web"
	"github.com/gorilla/websocket"
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

	// Now that we've joined the game, connect via WebSockets.
	wsClient, err := client.listenForUpdates(gameID)
	if err != nil {
		log.Fatalf("failed to listen for update: %v", err)
	}

	if *gameToJoin == "" {
		// Means we created the game, so we need to start it.
		reader := bufio.NewReader(os.Stdin)
		for {
			fmt.Print("Press ENTER to start game")
			_, _ = reader.ReadString('\n')
			if err := client.startGame(gameID); err != nil {
				log.Printf("failed to start game: %v", err)
				continue
			}
			break
		}
	} else {
		wsClient.waitForGameToStart()
	}
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

type wsClient struct {
	conn      *websocket.Conn
	done      chan struct{}
	gameStart chan *web.GameStart
}

func (ws *wsClient) read() {
	defer close(ws.done)
	for {
		messageType, message, err := ws.conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}
		if messageType != websocket.TextMessage {
			continue
		}

		var justAction struct {
			Action string `json:"action"`
		}
		if err := json.Unmarshal(message, &justAction); err != nil {
			log.Println("unmarshal:", err)
			return
		}

		switch justAction.Action {
		case "GAME_START":
			log.Println("GAME_START", string(message))
			ws.handleGameStart(message)
		case "CLUE_GIVEN":
			log.Println("CLUE_GIVEN", string(message))
		case "PLAYER_VOTE":
			log.Println("PLAYER_VOTE", string(message))
		case "GUESS_GIVEN":
			log.Println("GUESS_GIVEN", string(message))
		case "GAME_END":
			log.Println("GAME_END", string(message))
		}
	}
}

func (ws *wsClient) handleGameStart(dat []byte) {
	var gs web.GameStart
	if err := json.Unmarshal(dat, &gs); err != nil {
		log.Printf("handleGameStart: %v", err)
		return
	}

	ws.gameStart <- &gs
}

func (ws *wsClient) waitForGameToStart() {
	<-ws.gameStart
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
