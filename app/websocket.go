package main

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // tighten in production (same-origin, explicit origins)
	},
}

func runIndexHubWebSocket(w http.ResponseWriter, r *http.Request, a *App) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade: %v", err)
		return
	}

	a.indexHub.add(conn, "")
	a.broadcastLobbyState()

	go func() {
		defer func() {
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
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade: %v", err)
		return
	}

	if len(displayName) > 120 {
		displayName = displayName[:120]
	}
	h.add(conn, displayName)
	a.cancelRoomEviction(roomID)
	h.broadcastRoomState(nil)
	a.broadcastLobbyState()

	go func() {
		defer func() {
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
				switch action {
				case adminClearVotes:
					h.clearVotes()
				case adminAlwaysShowVotes:
					h.toggleAlwaysShowVotes()
				}
				h.broadcastRoomState(nil)
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
