package ws

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func dial(t *testing.T, server *httptest.Server) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func TestHub_BroadcastReachesConnectedClients(t *testing.T) {
	hub := NewHub()
	server := httptest.NewServer(hub)
	t.Cleanup(server.Close)

	conn := dial(t, server)

	// Give the server a moment to finish registering the client before we
	// broadcast, since the upgrade handshake and registration race with
	// this goroutine.
	time.Sleep(50 * time.Millisecond)

	hub.Broadcast(map[string]string{"event": "kickoff"})

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, raw, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}

	var got map[string]string
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal broadcast: %v", err)
	}
	if got["event"] != "kickoff" {
		t.Fatalf("got %v, want event=kickoff", got)
	}
}

func TestHub_BroadcastToMultipleClients(t *testing.T) {
	hub := NewHub()
	server := httptest.NewServer(hub)
	t.Cleanup(server.Close)

	conn1 := dial(t, server)
	conn2 := dial(t, server)
	time.Sleep(50 * time.Millisecond)

	hub.Broadcast(map[string]int{"score": 1})

	for _, conn := range []*websocket.Conn{conn1, conn2} {
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		if _, _, err := conn.ReadMessage(); err != nil {
			t.Fatalf("ReadMessage: %v", err)
		}
	}
}

func TestHub_DisconnectRemovesClient(t *testing.T) {
	hub := NewHub()
	server := httptest.NewServer(hub)
	t.Cleanup(server.Close)

	conn := dial(t, server)
	time.Sleep(50 * time.Millisecond)
	conn.Close()
	time.Sleep(50 * time.Millisecond)

	hub.mu.Lock()
	n := len(hub.clients)
	hub.mu.Unlock()
	if n != 0 {
		t.Fatalf("expected 0 clients after disconnect, got %d", n)
	}
}
