package main

import (
	"bytes"
	"log"
	"net/http"
	"sort"
)

// LobbyRoomRow is one row in the index lobby table (html/template requires exported fields).
type LobbyRoomRow struct {
	RoomID      string
	DisplayName string
	Count       int
}

// lobbyPageData is the index page (and live lobby OOB fragment) template data.
type lobbyPageData struct {
	LobbyCount     int
	RoomCount      int
	RoomsUserCount int
	Rooms          []LobbyRoomRow
	ListRooms      bool
	OOB            bool
}

func roomDisplayName(id, name string) string {
	if name == "" {
		return "Room " + id
	}
	return name + " · " + id
}

func (a *App) snapshotLobbyOverview(oob bool) lobbyPageData {
	a.mu.Lock()
	defer a.mu.Unlock()

	listRooms := a.listLobbyRooms
	lobbyCount := a.indexHub.count()
	ids := make([]string, 0, len(a.roomHubs))
	for id := range a.roomHubs {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	roomsUserCount := 0
	var rooms []LobbyRoomRow
	if listRooms {
		rooms = make([]LobbyRoomRow, 0, len(ids))
	}
	for _, id := range ids {
		cnt := a.roomHubs[id].count()
		roomsUserCount += cnt
		if listRooms {
			name := ""
			if room, ok := a.rooms[id]; ok {
				name = room.name
			}
			rooms = append(rooms, LobbyRoomRow{
				RoomID:      id,
				DisplayName: roomDisplayName(id, name),
				Count:       cnt,
			})
		}
	}
	return lobbyPageData{
		LobbyCount:     lobbyCount,
		RoomCount:      len(ids),
		RoomsUserCount: roomsUserCount,
		Rooms:          rooms,
		ListRooms:      listRooms,
		OOB:            oob,
	}
}

func (a *App) lobbyOverviewOOBHTML() string {
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "lobby-overview-table", a.snapshotLobbyOverview(true)); err != nil {
		log.Printf("lobby overview template: %v", err)
		return ""
	}
	return buf.String()
}

// broadcastLobbyState pushes the lobby overview table to everyone on the index page WebSocket.
func (a *App) broadcastLobbyState() {
	fragment := a.lobbyOverviewOOBHTML()
	a.indexHub.writeTextToAll([]byte(fragment))
}

func (a *App) home(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "index.html", a.snapshotLobbyOverview(false)); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(buf.Bytes())
}

func (a *App) indexWS(w http.ResponseWriter, r *http.Request) {
	runIndexHubWebSocket(w, r, a)
}
