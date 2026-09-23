package main

import (
	"crypto/rand"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

var cryptoReader io.Reader = rand.Reader

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
	name := truncateRunes(strings.TrimSpace(r.FormValue("name")), maxDisplayNameLen)

	a.mu.Lock()
	var id string
	for range 64 {
		candidate, err := randomSixDigitRoomID()
		if err != nil {
			a.mu.Unlock()
			log.Printf("room id: %v", err)
			http.Error(w, "could not allocate room", http.StatusServiceUnavailable)
			return
		}
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
	h := newRoomHub(name)
	a.roomHubs[id] = h
	a.scheduleRoomEvictionLocked(id, h)
	a.mu.Unlock()

	a.broadcastLobbyState()

	http.Redirect(w, r, "/"+id, http.StatusSeeOther)
}

func (a *App) roomPage(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomID")
	h, ok := a.getHub(roomID)
	if !ok {
		writeHTML(w, http.StatusNotFound, "room_not_found.html", struct{ RoomID string }{RoomID: roomID})
		return
	}

	data := struct {
		RoomID                string
		RoomName              string
		TopicTitle            string
		Count                 int
		ConsensusControlsHTML template.HTML
	}{
		RoomID:                roomID,
		RoomName:              h.name(),
		TopicTitle:            h.topic(),
		Count:                 h.count(),
		ConsensusControlsHTML: template.HTML(consensusControlsHTML(h.consensus(), h.allowedMaxSpread())),
	}
	writeHTML(w, http.StatusOK, "room.html", data)
}

func (a *App) roomWS(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomID")
	h, ok := a.getHub(roomID)
	if !ok {
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}
	runRoomHubWebSocket(w, r, a, roomID, h)
}

func randomSixDigitRoomID() (string, error) {
	n, err := rand.Int(cryptoReader, big.NewInt(900000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", int(n.Int64())+100000), nil
}
