package main

import (
	"os"
	"sync"
	"time"
)

const lobbyListRoomsEnv = "FIBO_PLANNER_LOBBY_LIST_ROOMS"

// App holds the home-page hub, per-room hubs, and optional display names for rooms created via the form.
type App struct {
	mu              sync.Mutex
	indexHub        *Hub // connections open on "/" (live session count on the index page)
	roomHubs        map[string]*Hub
	roomEvictTimers map[string]*time.Timer // pending idle-eviction per room
	rooms           map[string]*Room
	listLobbyRooms  bool // FIBO_PLANNER_LOBBY_LIST_ROOMS=Y lists each room on the lobby
}

func newApp() *App {
	return newAppConfig(os.Getenv(lobbyListRoomsEnv) == "Y")
}

func newAppConfig(listLobbyRooms bool) *App {
	return &App{
		indexHub:        newHub(),
		roomHubs:        make(map[string]*Hub),
		rooms:           make(map[string]*Room),
		roomEvictTimers: make(map[string]*time.Timer),
		listLobbyRooms:  listLobbyRooms,
	}
}

// lookupRoom returns the hub and display name for a live room under one lock.
// A hub without a matching rooms entry is treated as missing so callers never
// dereference a nil *Room after eviction.
func (a *App) lookupRoom(roomID string) (*Hub, string, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	h, ok := a.roomHubs[roomID]
	if !ok {
		return nil, "", false
	}
	room, ok := a.rooms[roomID]
	if !ok {
		return nil, "", false
	}
	return h, room.name, true
}

func (a *App) getHub(roomID string) (*Hub, bool) {
	h, _, ok := a.lookupRoom(roomID)
	return h, ok
}
