package main

import (
	"os"
	"sync"
	"time"
)

const lobbyListRoomsEnv = "FIBO_PLANNER_LOBBY_LIST_ROOMS"

// App holds the home-page connection set and per-room hubs (each hub owns its optional display name).
type App struct {
	mu              sync.Mutex
	indexConns      *connSet // connections open on "/" (live session count on the index page)
	roomHubs        map[string]*Hub
	roomEvictTimers map[string]*time.Timer // pending idle-eviction per room
	listLobbyRooms  bool                   // FIBO_PLANNER_LOBBY_LIST_ROOMS=Y lists each room on the lobby
}

func newApp() *App {
	return newAppConfig(os.Getenv(lobbyListRoomsEnv) == "Y")
}

func newAppConfig(listLobbyRooms bool) *App {
	return &App{
		indexConns:      newConnSet(),
		roomHubs:        make(map[string]*Hub),
		roomEvictTimers: make(map[string]*time.Timer),
		listLobbyRooms:  listLobbyRooms,
	}
}

func (a *App) getHub(roomID string) (*Hub, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	h, ok := a.roomHubs[roomID]
	return h, ok
}
