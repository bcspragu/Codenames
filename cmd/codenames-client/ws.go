package main

import (
	"encoding/json"
	"log"

	"github.com/bcspragu/Codenames/codenames"
	"github.com/bcspragu/Codenames/web"
	"github.com/gorilla/websocket"
)

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

func (ws *wsClient) waitForGameToStart() *web.GameStart {
	return <-ws.gameStart
}

func (ws *wsClient) waitForTurn(team codenames.Team, role codenames.Role) {
	// If this function was called, it's not our turn, so just show what's
	// happening until it is.
}
