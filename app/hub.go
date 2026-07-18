package main

import (
	"fmt"
	"html"
	"log"
	"sort"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

// Hub tracks active WebSocket connections (one browser tab/session) for one room.
type Hub struct {
	mu    sync.Mutex
	conns map[*websocket.Conn]string // display name per connection
}

func newHub() *Hub {
	return &Hub{conns: make(map[*websocket.Conn]string)}
}

func (h *Hub) add(c *websocket.Conn, displayName string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	if strings.TrimSpace(displayName) == "" {
		displayName = "Guest"
	}
	h.conns[c] = displayName
	return len(h.conns)
}

func (h *Hub) remove(c *websocket.Conn) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.conns, c)
	return len(h.conns)
}

func (h *Hub) count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.conns)
}

func (h *Hub) writeTextToAll(payload []byte) {
	h.mu.Lock()
	conns := make([]*websocket.Conn, 0, len(h.conns))
	for c := range h.conns {
		conns = append(conns, c)
	}
	h.mu.Unlock()
	for _, c := range conns {
		if err := c.WriteMessage(websocket.TextMessage, payload); err != nil {
			log.Printf("websocket write: %v", err)
		}
	}
}

// broadcastRoomState sends session count and the sorted list of participant names (room page only).
func (h *Hub) broadcastRoomState() {
	h.mu.Lock()
	n := len(h.conns)
	names := make([]string, 0, len(h.conns))
	for _, name := range h.conns {
		names = append(names, name)
	}
	conns := make([]*websocket.Conn, 0, len(h.conns))
	for c := range h.conns {
		conns = append(conns, c)
	}
	h.mu.Unlock()

	sort.Strings(names)
	var listHTML strings.Builder
	for _, name := range names {
		listHTML.WriteString("<li>")
		listHTML.WriteString(html.EscapeString(name))
		listHTML.WriteString("</li>")
	}

	fragment := fmt.Sprintf(
		`<strong id="session-count" hx-swap-oob="true">%d</strong>`+
			`<ul id="user-list" class="user-list" hx-swap-oob="true">%s</ul>`,
		n, listHTML.String(),
	)

	for _, c := range conns {
		if err := c.WriteMessage(websocket.TextMessage, []byte(fragment)); err != nil {
			log.Printf("websocket write: %v", err)
		}
	}
}
