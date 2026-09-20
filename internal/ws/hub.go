// Package ws implements a WebSocket hub that broadcasts live match events
// to every connected client.
package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Hub tracks connected clients and fans out broadcast events to all of them.
// It implements http.Handler: mount it directly at a WebSocket route.
type Hub struct {
	mu      sync.Mutex
	clients map[*client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[*client]struct{})}
}

// ServeHTTP upgrades the request to a WebSocket connection and registers it
// with the hub until the client disconnects.
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws: upgrade failed: %v", err)
		return
	}

	c := &client{conn: conn, send: make(chan []byte, 16)}
	h.register(c)

	go c.writePump()
	c.readPump(func() { h.unregister(c) })
}

func (h *Hub) register(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = struct{}{}
}

func (h *Hub) unregister(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		close(c.send)
	}
}

// Broadcast JSON-encodes event and sends it to every currently connected
// client. A client that can't keep up is dropped rather than allowed to
// block delivery to everyone else.
func (h *Hub) Broadcast(event any) {
	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf("ws: marshal event: %v", err)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		select {
		case c.send <- payload:
		default:
			log.Printf("ws: dropping slow client")
			delete(h.clients, c)
			close(c.send)
		}
	}
}
