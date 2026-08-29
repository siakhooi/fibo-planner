package main

import (
	"log"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

type participant struct {
	name     string
	points   string
	flash    bool
	observer bool
}

// Hub tracks active WebSocket connections (one browser tab/session) for one room.
type Hub struct {
	mu              sync.Mutex
	writeMu         sync.Mutex
	conns           map[*websocket.Conn]participant
	alwaysShowVotes bool
}

func newHub() *Hub {
	return &Hub{conns: make(map[*websocket.Conn]participant)}
}

func (h *Hub) add(c *websocket.Conn, displayName string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	if strings.TrimSpace(displayName) == "" {
		displayName = "Guest"
	}
	h.conns[c] = participant{name: displayName}
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

func (h *Hub) setPoints(c *websocket.Conn, points string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	p, ok := h.conns[c]
	if !ok || p.observer {
		return false
	}
	p.points = points
	h.conns[c] = p
	return true
}

func (h *Hub) toggleObserver(c *websocket.Conn) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	p, ok := h.conns[c]
	if !ok {
		return false
	}
	p.observer = !p.observer
	if p.observer {
		p.points = ""
	}
	h.conns[c] = p
	return true
}

func (h *Hub) clearVotes() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for c, p := range h.conns {
		p.points = ""
		h.conns[c] = p
	}
}

func (h *Hub) toggleAlwaysShowVotes() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.alwaysShowVotes = !h.alwaysShowVotes
}

func (h *Hub) writeTextToAll(payload []byte) {
	h.mu.Lock()
	conns := make([]*websocket.Conn, 0, len(h.conns))
	for c := range h.conns {
		conns = append(conns, c)
	}
	h.mu.Unlock()
	h.writeMu.Lock()
	defer h.writeMu.Unlock()
	for _, c := range conns {
		if err := c.WriteMessage(websocket.TextMessage, payload); err != nil {
			log.Printf("websocket write: %v", err)
		}
	}
}

// broadcastRoomState sends session count and the participant table (room page only).
func (h *Hub) broadcastRoomState(voted *websocket.Conn) {
	h.mu.Lock()
	n := len(h.conns)

	alwaysShow := h.alwaysShowVotes

	rows := make([]participant, 0, n)
	for c, p := range h.conns {
		p.flash = voted != nil && c == voted
		rows = append(rows, p)
	}

	h.mu.Unlock()
	h.writeTextToAll([]byte(roomStateHTML(n, rows, alwaysShow)))
}
