package hub

import (
	"fmt"
	"math/rand"

	"github.com/bcspragu/Codenames/db"
	"github.com/gorilla/websocket"
)

// Hub maintains the set of active connections and broadcasts messages to the
// connections.
type Hub struct {
	// Registered connections.
	connections map[db.GameID][]*connection

	// Inbound messages from the connections.
	broadcast chan *broadcastMsg

	// Register requests from the connections.
	register chan *connection

	// Unregister requests from connections.
	unregister chan *connection
}

// New creates a new Hub and starts it in a background Go routine.
func New() *Hub {
	h := &Hub{
		broadcast:   make(chan *broadcastMsg),
		register:    make(chan *connection),
		unregister:  make(chan *connection),
		connections: make(map[db.GameID][]*connection),
	}
	go h.run()
	return h
}

func (h *Hub) run() {
	for {
		select {
		case c := <-h.register:
			conns := h.connections[c.gameID]
			h.connections[c.gameID] = append(conns, c)
		case c := <-h.unregister:
			h.deleteConn(c)
		case m := <-h.broadcast:
			for _, c := range h.connections[m.gameID] {
				select {
				case c.send <- m.msg:
				default:
					h.deleteConn(c)
				}
			}
		}
	}
}

func (h *Hub) deleteConn(c *connection) {
	close(c.send)
	rconns := h.connections[c.gameID]
	for i, rconn := range rconns {
		if rconn.id == c.id {
			// Remove the connection.
			copy(rconns[i:], rconns[i+1:])
			rconns[len(rconns)-1] = nil
			h.connections[c.gameID] = rconns[:len(rconns)-1]
			return
		}
	}
}

type broadcastMsg struct {
	gameID db.GameID
	msg    []byte
}

// BroadcastGame sends a message to everyone in a game.
func (h *Hub) BroadcastGame(msg []byte, gID db.GameID) {
	h.broadcast <- &broadcastMsg{
		gameID: gID,
		msg:    msg,
	}
}

// Register associates a connection with the hub and a given game.
func (h *Hub) Register(ws *websocket.Conn, gID db.GameID) {
	conn := &connection{id: newID(gID), h: h, gameID: gID, send: make(chan []byte, 256), ws: ws}
	h.register <- conn
	go conn.writePump()
	go conn.readPump()
}

func newID(gID db.GameID) string {
	return fmt.Sprintf("%s-%d", gID, rand.Int63())
}
