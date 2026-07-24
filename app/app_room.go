package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// roomIdleEvictionDelay is how long a room with zero WebSocket connections may stay before it is removed.
const roomIdleEvictionDelay = 30 * time.Minute

// scheduleRoomEvictionLocked starts (or replaces) the idle timer for an empty room. Caller must hold a.mu.
func (a *App) scheduleRoomEvictionLocked(roomID string, h *Hub) {
	if t, ok := a.roomEvictTimers[roomID]; ok {
		t.Stop()
		delete(a.roomEvictTimers, roomID)
	}
	timer := time.AfterFunc(roomIdleEvictionDelay, func() {
		a.evictRoomIfStillEmpty(roomID, h)
	})
	a.roomEvictTimers[roomID] = timer
}

func (a *App) scheduleRoomEviction(roomID string, h *Hub) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.scheduleRoomEvictionLocked(roomID, h)
}

func (a *App) cancelRoomEviction(roomID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if t, ok := a.roomEvictTimers[roomID]; ok {
		t.Stop()
		delete(a.roomEvictTimers, roomID)
	}
}

func (a *App) evictRoomIfStillEmpty(roomID string, h *Hub) {
	a.mu.Lock()
	current, ok := a.roomHubs[roomID]
	if !ok || current != h {
		a.mu.Unlock()
		return
	}
	if h.count() != 0 {
		delete(a.roomEvictTimers, roomID)
		a.mu.Unlock()
		return
	}
	delete(a.roomHubs, roomID)
	delete(a.rooms, roomID)
	delete(a.roomEvictTimers, roomID)
	a.mu.Unlock()

	log.Printf("removed room after %v idle: %s", roomIdleEvictionDelay, roomID)
	a.broadcastLobbyState()
}

func (a *App) createRoom(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if len(name) > 120 {
		name = name[:120]
	}

	a.mu.Lock()
	var id string
	for range 64 {
		candidate := randomSixDigitRoomID()
		if _, exists := a.roomHubs[candidate]; !exists {
			id = candidate
			break
		}
	}
	if id == "" {
		a.mu.Unlock()
		http.Error(w, "could not allocate room", http.StatusServiceUnavailable)
		return
	}
	a.roomHubs[id] = newHub()
	if name != "" {
		a.rooms[id] = newRoom(name)
	}
	a.scheduleRoomEvictionLocked(id, a.roomHubs[id])
	a.mu.Unlock()

	a.broadcastLobbyState()

	http.Redirect(w, r, "/"+id, http.StatusSeeOther)
}

func (a *App) roomPage(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomID")
	h, ok := a.getHub(roomID)
	if !ok {
		data := struct{ RoomID string }{RoomID: roomID}
		var buf bytes.Buffer
		if err := tmpl.ExecuteTemplate(&buf, "room_not_found.html", data); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write(buf.Bytes())
		return
	}

	a.mu.Lock()
	name := a.rooms[roomID].name
	a.mu.Unlock()

	data := struct {
		RoomID   string
		RoomName string
		Count    int
	}{
		RoomID:   roomID,
		RoomName: name,
		Count:    h.count(),
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "room.html", data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(buf.Bytes())
}

func (a *App) roomWS(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomID")
	h, ok := a.getHub(roomID)
	if !ok {
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	runRoomHubWebSocket(w, r, a, roomID, h, name)
}

func randomSixDigitRoomID() string {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "100000"
	}
	return fmt.Sprintf("%06d", int(n.Int64())+100000)
}
