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
	topicTitle      string
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

func (h *Hub) clearVotesLocked() {
	for c, p := range h.conns {
		p.points = ""
		h.conns[c] = p
	}
}

func (h *Hub) resetTopic(title string, clearVotes bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(title) > 120 {
		title = title[:120]
	}
	h.topicTitle = title
	if clearVotes {
		h.clearVotesLocked()
	}
}

func (h *Hub) topic() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.topicTitle
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
// highlight, if non-nil, is the connection whose row should flash (vote, join, or role change).
func (h *Hub) broadcastRoomState(highlight *websocket.Conn) {
	h.mu.Lock()
	n := len(h.conns)

	alwaysShow := h.alwaysShowVotes
	topic := h.topicTitle

	rows := make([]participant, 0, n)
	for c, p := range h.conns {
		p.flash = highlight != nil && c == highlight
		rows = append(rows, p)
	}

	h.mu.Unlock()
	h.writeTextToAll([]byte(roomStateHTML(n, rows, alwaysShow, topic)))
}
