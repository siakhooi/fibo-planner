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

func writeWSPing(writeMu *sync.Mutex, conn *websocket.Conn) error {
	writeMu.Lock()
	defer writeMu.Unlock()
	return writeWS(conn, websocket.PingMessage, nil)
}

func startWSPing(writeMu *sync.Mutex, conn *websocket.Conn) func() {
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
				if err := writeWSPing(writeMu, conn); err != nil {
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

	a.indexConns.add(conn)
	a.broadcastLobbyState()

	go func() {
		stopPing := startWSPing(&a.indexConns.writeMu, conn)
		defer func() {
			stopPing()
			_ = conn.Close()
			a.indexConns.remove(conn)
			a.broadcastLobbyState()
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
}

func runRoomHubWebSocket(w http.ResponseWriter, r *http.Request, a *App, roomID string, h *Hub) {
	conn, err := acceptWebSocket(w, r)
	if err != nil {
		log.Printf("websocket upgrade: %v", err)
		return
	}

	a.cancelRoomEviction(roomID)

	go func() {
		stopPing := startWSPing(&h.writeMu, conn)
		joined := false
		defer func() {
			stopPing()
			_ = conn.Close()
			remaining := h.remove(conn)
			if joined {
				h.broadcastRoomState(nil)
				a.broadcastLobbyState()
			}
			if remaining == 0 {
				a.scheduleRoomEviction(roomID, h)
			}
		}()
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			m, ok := parseWSMessage(msg)
			if !ok {
				continue
			}
			if !joined {
				name, ok := m.joinName()
				if !ok {
					continue
				}
				h.add(conn, name)
				a.cancelRoomEviction(roomID)
				h.broadcastRoomState(conn)
				a.broadcastLobbyState()
				joined = true
				continue
			}
			if action, isAdmin := m.adminAction(); isAdmin {
				var highlight *websocket.Conn
				switch action {
				case adminClearVotes:
					h.clearVotes()
				case adminSetTopic:
					h.setTopic(m.topicTitle())
				case adminLoadNextTopic:
					h.loadNextTopic()
				case adminSetPreloadedTopics:
					h.setPreloadedTopics(m.preloadedTopics())
				case adminAlwaysShowVotes:
					h.toggleAlwaysShowVotes()
				case adminConsensusAgreement:
					if p, ok := m.consensusPercent(); ok {
						h.setConsensusPercent(p)
					}
					if s, ok := m.maxSpread(); ok {
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
			points, ok := m.votePoints()
			if !ok {
				continue
			}
			if h.setPoints(conn, points) {
				h.broadcastRoomState(conn)
			}
		}
	}()
}
