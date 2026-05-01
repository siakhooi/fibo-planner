package main

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"
)

//go:embed index.html
var indexFS embed.FS

var indexTpl = template.Must(template.ParseFS(indexFS, "index.html"))

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // tighten in production (same-origin, explicit origins)
	},
}

// Hub tracks active WebSocket connections (one per browser session).
type Hub struct {
	mu    sync.Mutex
	conns map[*websocket.Conn]struct{}
}

func newHub() *Hub {
	return &Hub{conns: make(map[*websocket.Conn]struct{})}
}

func (h *Hub) add(c *websocket.Conn) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.conns[c] = struct{}{}
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

// broadcastCount sends an HTMX out-of-band swap so every client updates the counter.
func (h *Hub) broadcastCount() {
	n := h.count()
	fragment := fmt.Sprintf(
		`<strong id="session-count" hx-swap-oob="true">%d</strong>`,
		n,
	)

	h.mu.Lock()
	conns := make([]*websocket.Conn, 0, len(h.conns))
	for c := range h.conns {
		conns = append(conns, c)
	}
	h.mu.Unlock()

	for _, c := range conns {
		if err := c.WriteMessage(websocket.TextMessage, []byte(fragment)); err != nil {
			log.Printf("websocket write: %v", err)
		}
	}
}

func main() {
	hub := newHub()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		data := struct{ Count int }{Count: hub.count()}
		if err := indexTpl.Execute(&buf, data); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(buf.Bytes())
	})

	r.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("websocket upgrade: %v", err)
			return
		}

		hub.add(conn)
		hub.broadcastCount()

		go func() {
			defer func() {
				_ = conn.Close()
				hub.remove(conn)
				hub.broadcastCount()
			}()
			// Read until the client disconnects (required to observe close / ping handling).
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
			}
		}()
	})

	addr := ":8080"
	log.Printf("listening on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}
