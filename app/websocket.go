package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	wsOriginsEnv        = "FIBO_PLANNER_WS_ORIGINS"
	maxWSMessageBytes   = 128 * 1024
	defaultWSPongWait   = 60 * time.Second
	defaultWSPingPeriod = (defaultWSPongWait * 9) / 10
	defaultWSWriteWait  = 10 * time.Second
)

var (
	wsPongWait   = defaultWSPongWait
	wsPingPeriod = defaultWSPingPeriod
	wsWriteWait  = defaultWSWriteWait
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     checkWSOrigin,
}

func parseWSOrigins(v string) []string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimRight(strings.TrimSpace(p), "/")
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func checkWSOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	if strings.EqualFold(u.Host, r.Host) {
		return true
	}
	normalized := strings.TrimRight(origin, "/")
	for _, allowed := range parseWSOrigins(os.Getenv(wsOriginsEnv)) {
		if strings.EqualFold(normalized, allowed) {
			return true
		}
	}
	return false
}

func writeWS(c *websocket.Conn, messageType int, payload []byte) error {
	_ = c.SetWriteDeadline(time.Now().Add(wsWriteWait))
	return c.WriteMessage(messageType, payload)
}

func prepareWebSocket(conn *websocket.Conn) {
	conn.SetReadLimit(maxWSMessageBytes)
	_ = conn.SetReadDeadline(time.Now().Add(wsPongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(wsPongWait))
	})
}

func (h *Hub) writePing(conn *websocket.Conn) error {
	h.writeMu.Lock()
	defer h.writeMu.Unlock()
	return writeWS(conn, websocket.PingMessage, nil)
}

func (h *Hub) startPing(conn *websocket.Conn) func() {
	done := make(chan struct{})
	var once sync.Once
	go func() {
		ticker := time.NewTicker(wsPingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if err := h.writePing(conn); err != nil {
					_ = conn.Close()
					return
				}
			}
		}
	}()
	return func() {
		once.Do(func() { close(done) })
	}
}

func acceptWebSocket(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	prepareWebSocket(conn)
	return conn, nil
}

func runIndexHubWebSocket(w http.ResponseWriter, r *http.Request, a *App) {
	conn, err := acceptWebSocket(w, r)
	if err != nil {
		log.Printf("websocket upgrade: %v", err)
		return
	}

	a.indexHub.add(conn, "")
	a.broadcastLobbyState()

	go func() {
		stopPing := a.indexHub.startPing(conn)
		defer func() {
			stopPing()
			_ = conn.Close()
			a.indexHub.remove(conn)
			a.broadcastLobbyState()
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
}

func runRoomHubWebSocket(w http.ResponseWriter, r *http.Request, a *App, roomID string, h *Hub, displayName string) {
	conn, err := acceptWebSocket(w, r)
	if err != nil {
		log.Printf("websocket upgrade: %v", err)
		return
	}

	h.add(conn, displayName)
	a.cancelRoomEviction(roomID)
	h.broadcastRoomState(conn)
	a.broadcastLobbyState()

	go func() {
		stopPing := h.startPing(conn)
		defer func() {
			stopPing()
			_ = conn.Close()
			remaining := h.remove(conn)
			h.broadcastRoomState(nil)
			a.broadcastLobbyState()
			if remaining == 0 {
				a.scheduleRoomEviction(roomID, h)
			}
		}()
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			action, isAdmin := parseAdminAction(msg)
			if isAdmin {
				var highlight *websocket.Conn
				switch action {
				case adminClearVotes:
					h.clearVotes()
				case adminSetTopic:
					h.setTopic(parseTopicTitle(msg))
				case adminLoadNextTopic:
					h.loadNextTopic()
				case adminSetPreloadedTopics:
					h.setPreloadedTopics(parsePreloadedTopics(msg))
				case adminAlwaysShowVotes:
					h.toggleAlwaysShowVotes()
				case adminConsensusAgreement:
					if p, ok := parseConsensusPercent(msg); ok {
						h.setConsensusPercent(p)
					}
					if s, ok := parseMaxSpread(msg); ok {
						h.setMaxSpread(s)
					}
				case adminObserverMode:
					if h.toggleObserver(conn) {
						highlight = conn
					}
				}
				h.broadcastRoomState(highlight)
				continue
			}
			points, ok := parseVotePoints(msg)
			if !ok {
				continue
			}
			if h.setPoints(conn, points) {
				h.broadcastRoomState(conn)
			}
		}
	}()
}
