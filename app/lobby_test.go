package main

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

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

func TestLobbySnapshotUnnamedHubUsesRoomID(t *testing.T) {
	a := newAppConfig(true)
	a.roomHubs["123456"] = newRoomHub("")

	got := a.snapshotLobbyOverview(false)
	if got.RoomCount != 1 {
		t.Fatalf("RoomCount=%d, want 1", got.RoomCount)
	}
	if len(got.Rooms) != 1 {
		t.Fatalf("listed %d rooms, want 1", len(got.Rooms))
	}
	if got.Rooms[0].DisplayName != "Room 123456" {
		t.Fatalf("DisplayName=%q, want unnamed fallback", got.Rooms[0].DisplayName)
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
	a := newAppConfig(false)
	srv := httptest.NewServer(newRouter(a))
	t.Cleanup(srv.Close)

	first := createRoom(t, srv, "alpha")
	second := createRoom(t, srv, "beta")
	dialRoom(t, srv, first, "Ada")
	dialRoom(t, srv, first, "Bob")
	dialRoom(t, srv, second, "Cyd")

	deadline := time.Now().Add(time.Second)
	for a.snapshotLobbyOverview(false).RoomsUserCount != 3 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := a.snapshotLobbyOverview(false).RoomsUserCount; got != 3 {
		t.Fatalf("joined users=%d, want 3", got)
	}

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

func TestLobbyOverviewFragmentTemplateError(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("stale")
	if got := lobbyOverviewFragment(&buf, errors.New("boom")); got != "" {
		t.Fatalf("template error should yield an empty fragment, got %q", got)
	}
}

func TestIndexWSBroadcastsLobbyOnConnect(t *testing.T) {
	a := newAppConfig(false)
	seedLobbyRoom(a, "111111", "sprint", 1)
	srv := httptest.NewServer(newRouter(a))
	t.Cleanup(srv.Close)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	msg := waitForMessage(t, conn, `id="lobby-overview"`)
	if !strings.Contains(msg, `hx-swap-oob="true"`) {
		t.Fatalf("connect broadcast missing OOB marker: %s", msg)
	}
	if !strings.Contains(msg, `id="session-count">1</strong>`) {
		t.Fatalf("connect broadcast missing lobby count: %s", msg)
	}
	if !strings.Contains(msg, `id="room-count">1</strong>`) || !strings.Contains(msg, `id="rooms-user-count">1</strong>`) {
		t.Fatalf("connect broadcast missing totals: %s", msg)
	}
	if strings.Contains(msg, "sprint") || strings.Contains(msg, `href="/111111"`) {
		t.Fatalf("connect broadcast should hide the room list: %s", msg)
	}
	if a.indexConns.count() != 1 {
		t.Fatalf("lobby connections=%d, want 1", a.indexConns.count())
	}

	_ = conn.Close()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if a.indexConns.count() == 0 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("lobby socket was not removed")
}

func seedLobbyRoom(a *App, id, name string, users int) {
	h := newRoomHub(name)
	for range users {
		h.add(&websocket.Conn{}, "guest")
	}
	a.roomHubs[id] = h
}
