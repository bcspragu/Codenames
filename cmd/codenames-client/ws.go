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
		conn:  conn,
		done:  make(chan struct{}),
		hooks: hooks,
	}

	return wsc.read()
}

func (ws *wsClient) read() error {
	defer close(ws.done)
	for {
		messageType, message, err := ws.conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("ReadMessage: %w", err)
		}
		if messageType == websocket.PingMessage {
			if err := ws.conn.WriteMessage(websocket.PongMessage, []byte{}); err != nil {
				return fmt.Errorf("failed to send pong: %w", err)
			}
		}

		if messageType != websocket.TextMessage {
			continue
		}

		var justAction struct {
			Action string `json:"action"`
		}
		if err := json.Unmarshal(message, &justAction); err != nil {
			return fmt.Errorf("json.Unmarshal: %w", err)
		}

		switch justAction.Action {
		case "GAME_START":
			ws.handleGameStart(message)
		case "CLUE_GIVEN":
			ws.handleClueGiven(message)
		case "PLAYER_VOTE":
			ws.handlePlayerVote(message)
		case "GUESS_GIVEN":
			ws.handleGuessGiven(message)
		case "GAME_END":
			ws.handleGameEnd(message)
		default:
			log.Printf("unknown message action %q", justAction.Action)
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
