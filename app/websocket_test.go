package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestRoomPageHasPointsTable(t *testing.T) {
	srv := httptest.NewServer(newRouter(newApp()))
	t.Cleanup(srv.Close)

	roomID := createRoom(t, srv, "sprint")
	resp, err := http.Get(srv.URL + "/" + roomID)
	if err != nil {
		t.Fatalf("room page: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	page := string(body)
	for _, want := range []string{
		`id="user-list"`,
		`class="user-table"`,
		`scope="col">Points`,
		`id="points-form"`,
		`data-points="8"`,
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("room page missing %q", want)
		}
	}
	if strings.Contains(page, `class="user-list"`) {
		t.Fatal("room page still has the old user-list ul")
	}
}

func TestVoteBroadcastToAllParticipants(t *testing.T) {
	srv := httptest.NewServer(newRouter(newApp()))
	t.Cleanup(srv.Close)

	roomID := createRoom(t, srv, "sprint")
	ada := dialRoom(t, srv, roomID, "Ada")
	waitForMessage(t, ada, "<td>Ada</td><td></td>")

	bob := dialRoom(t, srv, roomID, "Bob")
	waitForMessage(t, ada, "<td>Bob</td><td></td>")
	waitForMessage(t, bob, "<td>Ada</td><td></td>")

	if err := ada.WriteMessage(websocket.TextMessage, []byte(`{"points":"8"}`)); err != nil {
		t.Fatalf("ada vote: %v", err)
	}
	gotAda := waitForMessage(t, ada, "<td>Ada</td><td>8</td>")
	gotBob := waitForMessage(t, bob, "<td>Ada</td><td>8</td>")
	if !strings.Contains(gotAda, `th scope="col">Points`) {
		t.Fatalf("table missing Points column: %s", gotAda)
	}
	if !strings.Contains(gotBob, "<td>Bob</td><td></td>") {
		t.Fatalf("bob row should still have empty points: %s", gotBob)
	}
}

func createRoom(t *testing.T, srv *httptest.Server, name string) string {
	t.Helper()
	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.PostForm(srv.URL+"/rooms", url.Values{"name": {name}})
	if err != nil {
		t.Fatalf("create room: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	loc := resp.Header.Get("Location")
	id := strings.TrimPrefix(loc, "/")
	if len(id) != 6 {
		t.Fatalf("unexpected room location %q", loc)
	}
	return id
}

func dialRoom(t *testing.T, srv *httptest.Server, roomID, name string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/" + roomID + "?name=" + url.QueryEscape(name)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial %s: %v", name, err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func waitForMessage(t *testing.T, conn *websocket.Conn, substr string) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	if err := conn.SetReadDeadline(deadline); err != nil {
		t.Fatalf("deadline: %v", err)
	}
	var last string
	for time.Now().Before(deadline) {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read waiting for %q: %v (last=%q)", substr, err, last)
		}
		last = string(msg)
		if strings.Contains(last, substr) {
			return last
		}
	}
	t.Fatalf("timeout waiting for %q (last=%q)", substr, last)
	return ""
}
