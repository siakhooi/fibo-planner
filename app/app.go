package main

import (
	"sync"
	"time"
)

// App holds the home-page hub, per-room hubs, and optional display names for rooms created via the form.
type App struct {
	mu              sync.Mutex
	indexHub        *Hub // connections open on "/" (live session count on the index page)
	roomHubs        map[string]*Hub
	roomEvictTimers map[string]*time.Timer // pending idle-eviction per room
	rooms           map[string]*Room
}

func newApp() *App {
	return &App{
		indexHub:        newHub(),
		roomHubs:        make(map[string]*Hub),
		rooms:           make(map[string]*Room),
		roomEvictTimers: make(map[string]*time.Timer),
	}
}

func (a *App) getHub(roomID string) (*Hub, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	h, ok := a.roomHubs[roomID]
	return h, ok
}
