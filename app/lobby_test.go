package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestNewAppLobbyListRoomsEnv(t *testing.T) {
	cases := []struct {
		val  string
		want bool
	}{
		{val: "", want: false},
		{val: "N", want: false},
		{val: "Y", want: true},
		{val: "y", want: false},
		{val: "yes", want: false},
	}
	for _, tc := range cases {
		t.Run("env="+tc.val, func(t *testing.T) {
			t.Setenv(lobbyListRoomsEnv, tc.val)
			a := newApp()
			if a.listLobbyRooms != tc.want {
				t.Fatalf("listLobbyRooms=%v, want %v", a.listLobbyRooms, tc.want)
			}
		})
	}
}

func TestLobbyHomeShowsTotalsWithoutRoomList(t *testing.T) {
	a := newAppConfig(false)
	seedLobbyRoom(a, "111111", "sprint", 2)
	seedLobbyRoom(a, "222222", "", 1)
	srv := httptest.NewServer(newRouter(a))
	t.Cleanup(srv.Close)

	page := getHTML(t, srv, "/", http.StatusOK)
	if !strings.Contains(page, `id="session-count">0</strong>`) {
		t.Fatalf("missing lobby count: %s", page)
	}
	if !strings.Contains(page, `id="room-count">2</strong>`) {
		t.Fatalf("missing room count: %s", page)
	}
	if !strings.Contains(page, `id="rooms-user-count">3</strong>`) {
		t.Fatalf("missing people in rooms: %s", page)
	}
	if strings.Contains(page, "sprint") || strings.Contains(page, "Room 111111") || strings.Contains(page, `href="/222222"`) {
		t.Fatalf("room list should be hidden: %s", page)
	}
	if strings.Contains(page, `hx-swap-oob="true"`) {
		t.Fatal("initial lobby table should not be an OOB swap")
	}
}

func TestLobbyHomeListsRoomsWhenEnabled(t *testing.T) {
	a := newAppConfig(true)
	seedLobbyRoom(a, "111111", "sprint", 2)
	seedLobbyRoom(a, "222222", "", 0)
	srv := httptest.NewServer(newRouter(a))
	t.Cleanup(srv.Close)

	page := getHTML(t, srv, "/", http.StatusOK)
	if !strings.Contains(page, `id="room-count">2</strong>`) {
		t.Fatalf("missing room count: %s", page)
	}
	if !strings.Contains(page, `id="rooms-user-count">2</strong>`) {
		t.Fatalf("missing people in rooms: %s", page)
	}
	if !strings.Contains(page, `<a href="/111111">sprint · 111111</a>`) {
		t.Fatalf("named room missing: %s", page)
	}
	if !strings.Contains(page, `<a href="/222222">Room 222222</a>`) {
		t.Fatalf("unnamed room missing: %s", page)
	}
}

func TestLobbyOverviewOOBRespectsRoomListFlag(t *testing.T) {
	hidden := newAppConfig(false)
	seedLobbyRoom(hidden, "111111", "sprint", 1)
	got := hidden.lobbyOverviewOOBHTML()
	if !strings.Contains(got, `id="lobby-overview"`) || !strings.Contains(got, `hx-swap-oob="true"`) {
		t.Fatalf("OOB table missing swap marker: %s", got)
	}
	if !strings.Contains(got, `id="room-count">1</strong>`) || !strings.Contains(got, `id="rooms-user-count">1</strong>`) {
		t.Fatalf("OOB totals missing: %s", got)
	}
	if strings.Contains(got, "sprint") || strings.Contains(got, `href="/111111"`) {
		t.Fatalf("OOB should not list rooms by default: %s", got)
	}

	listed := newAppConfig(true)
	seedLobbyRoom(listed, "111111", "sprint", 1)
	got = listed.lobbyOverviewOOBHTML()
	if !strings.Contains(got, `<a href="/111111">sprint · 111111</a>`) {
		t.Fatalf("OOB should list rooms when enabled: %s", got)
	}
}

func TestLobbyHomePeopleCountIncludesConnectedUsers(t *testing.T) {
	srv := httptest.NewServer(newRouter(newAppConfig(false)))
	t.Cleanup(srv.Close)

	first := createRoom(t, srv, "alpha")
	second := createRoom(t, srv, "beta")
	dialRoom(t, srv, first, "Ada")
	dialRoom(t, srv, first, "Bob")
	dialRoom(t, srv, second, "Cyd")

	page := getHTML(t, srv, "/", http.StatusOK)
	if !strings.Contains(page, `id="room-count">2</strong>`) {
		t.Fatalf("expected 2 rooms: %s", page)
	}
	if !strings.Contains(page, `id="rooms-user-count">3</strong>`) {
		t.Fatalf("expected 3 people in rooms: %s", page)
	}
	if strings.Contains(page, "alpha") || strings.Contains(page, "beta") {
		t.Fatalf("room names should stay hidden: %s", page)
	}
}

func seedLobbyRoom(a *App, id, name string, users int) {
	h := newHub()
	for range users {
		h.add(&websocket.Conn{}, "guest")
	}
	a.roomHubs[id] = h
	a.rooms[id] = newRoom(name)
}
