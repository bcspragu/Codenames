package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/bcspragu/Codenames/web"
	"github.com/gorilla/websocket"
)

type wsClient struct {
	conn  *websocket.Conn
	msgs  chan []byte
	done  chan struct{}
	hooks wsHooks
}

func (c *client) listenForUpdates(gID string, hooks wsHooks) error {
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
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	if hooks.onConnect != nil {
		go hooks.onConnect()
	}

	wsc := &wsClient{
		conn: conn,
		done: make(chan struct{}),
		// We buffer it in case messages come in while we're waiting on user input.
		// We don't want to process messages concurrently, because that seems
		// likely to cause tricky problems.
		msgs:  make(chan []byte, 100),
		hooks: hooks,
	}

	go wsc.handleMessages()

	return wsc.read()
}

func (ws *wsClient) read() error {
	defer close(ws.done)
	for {
		messageType, message, err := ws.conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("ReadMessage: %w", err)
		}

		if messageType != websocket.TextMessage {
			continue
		}

		ws.msgs <- message
	}
}

func (ws *wsClient) handleMessages() {
	for {
		select {
		case <-ws.done:
			return
		case msg := <-ws.msgs:
			var justAction struct {
				Action string `json:"action"`
			}
			if err := json.Unmarshal(msg, &justAction); err != nil {
				log.Printf("failed to unmarshal action from server: %v", err)
				return
			}

			switch justAction.Action {
			case "GAME_START":
				ws.handleGameStart(msg)
			case "CLUE_GIVEN":
				ws.handleClueGiven(msg)
			case "PLAYER_VOTE":
				ws.handlePlayerVote(msg)
			case "GUESS_GIVEN":
				ws.handleGuessGiven(msg)
			case "GAME_END":
				ws.handleGameEnd(msg)
			default:
				log.Printf("unknown message action %q", justAction.Action)
			}
		}
	}
}

func (ws *wsClient) handleGameStart(dat []byte) {
	var gs web.GameStart
	if err := json.Unmarshal(dat, &gs); err != nil {
		log.Printf("handleGameStart: %v", err)
		return
	}

	fmt.Println(string(dat))
	fmt.Printf("%+v\n", gs)
	if ws.hooks.onStart == nil {
		return
	}
	ws.hooks.onStart(&gs)
}

func (ws *wsClient) handleClueGiven(dat []byte) {
	var cg web.ClueGiven
	if err := json.Unmarshal(dat, &cg); err != nil {
		log.Printf("handleClueGiven: %v", err)
		return
	}

	if ws.hooks.onClueGiven == nil {
		return
	}
	ws.hooks.onClueGiven(&cg)
}

func (ws *wsClient) handlePlayerVote(dat []byte) {
	var pv web.PlayerVote
	if err := json.Unmarshal(dat, &pv); err != nil {
		log.Printf("handlePlayerVote: %v", err)
		return
	}

	if ws.hooks.onPlayerVote == nil {
		return
	}
	ws.hooks.onPlayerVote(&pv)
}

func (ws *wsClient) handleGuessGiven(dat []byte) {
	var gg web.GuessGiven
	if err := json.Unmarshal(dat, &gg); err != nil {
		log.Printf("handleGuessGiven: %v", err)
		return
	}

	if ws.hooks.onGuessGiven == nil {
		return
	}
	ws.hooks.onGuessGiven(&gg)
}

func (ws *wsClient) handleGameEnd(dat []byte) {
	var ge web.GameEnd
	if err := json.Unmarshal(dat, &ge); err != nil {
		log.Printf("handleGameEnd: %v", err)
		return
	}

	if ws.hooks.onEnd == nil {
		return
	}
	ws.hooks.onEnd(&ge)
}

type wsHooks struct {
	onConnect    func()
	onStart      func(*web.GameStart)
	onClueGiven  func(*web.ClueGiven)
	onPlayerVote func(*web.PlayerVote)
	onGuessGiven func(*web.GuessGiven)
	onEnd        func(*web.GameEnd)
}
