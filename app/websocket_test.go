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
		`Administration`,
		`id="always-show-votes"`,
		`id="clear-votes"`,
		`id="observer-mode"`,
		`class="results-panel"`,
		`id="vote-results"`,
		`scope="col">Count`,
		`scope="col">%`,
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("room page missing %q", want)
		}
	}
	if strings.Contains(page, `class="user-list"`) {
		t.Fatal("room page still has the old user-list ul")
	}
}

func TestCreateRoomWithoutName(t *testing.T) {
	srv := httptest.NewServer(newRouter(newApp()))
	t.Cleanup(srv.Close)

	roomID := createRoom(t, srv, "")

	home, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("home: %v", err)
	}
	defer func() { _ = home.Body.Close() }()
	body, err := io.ReadAll(home.Body)
	if err != nil {
		t.Fatalf("read home: %v", err)
	}
	if !strings.Contains(string(body), "Room "+roomID) {
		t.Fatalf("lobby missing unnamed room %s: %s", roomID, body)
	}

	room, err := http.Get(srv.URL + "/" + roomID)
	if err != nil {
		t.Fatalf("room page: %v", err)
	}
	defer func() { _ = room.Body.Close() }()
	if room.StatusCode != http.StatusOK {
		t.Fatalf("room page status %d", room.StatusCode)
	}
}
func TestVoteBroadcastToAllParticipants(t *testing.T) {
	srv := httptest.NewServer(newRouter(newApp()))
	t.Cleanup(srv.Close)

	roomID := createRoom(t, srv, "sprint")
	ada := dialRoom(t, srv, roomID, "Ada")
	waitForMessage(t, ada, `<td class="vote-flash">Ada</td><td class="vote-flash"></td>`)

	bob := dialRoom(t, srv, roomID, "Bob")
	joined := waitForMessage(t, ada, `<td class="vote-flash">Bob</td><td class="vote-flash"></td>`)
	if !strings.Contains(joined, "<td>Ada</td><td></td>") {
		t.Fatalf("Ada should not flash when Bob joins: %s", joined)
	}
	waitForMessage(t, bob, "<td>Ada</td><td></td>")

	if err := ada.WriteMessage(websocket.TextMessage, []byte(`{"points":"8"}`)); err != nil {
		t.Fatalf("ada vote: %v", err)
	}
	gotAda := waitForMessage(t, ada, `<td class="vote-flash">Ada</td><td class="vote-flash">???</td>`)
	gotBob := waitForMessage(t, bob, `<td class="vote-flash">Ada</td><td class="vote-flash">???</td>`)
	if !strings.Contains(gotAda, `th scope="col">Points`) {
		t.Fatalf("table missing Points column: %s", gotAda)
	}
	if !strings.Contains(gotBob, "<td>Bob</td><td></td>") {
		t.Fatalf("bob row should still have empty points: %s", gotBob)
	}
	if !strings.Contains(gotAda, `id="vote-results" class="user-table results-table" hx-swap-oob="true" hidden`) {
		t.Fatalf("results should stay hidden until everyone voted: %s", gotAda)
	}

	if err := bob.WriteMessage(websocket.TextMessage, []byte(`{"points":"5"}`)); err != nil {
		t.Fatalf("bob vote: %v", err)
	}
	revealed := waitForMessage(t, ada, "<td>Ada</td><td>8</td>")
	if !strings.Contains(revealed, `<td class="vote-flash">Bob</td><td class="vote-flash">5</td>`) {
		t.Fatalf("bob's vote should be highlighted: %s", revealed)
	}
	if strings.Contains(revealed, `id="vote-results" class="user-table results-table" hx-swap-oob="true" hidden`) {
		t.Fatalf("results should be visible once everyone voted: %s", revealed)
	}
	five := strings.Index(revealed, `<tr class="vote-leader"><td>5</td><td>1</td><td>50%</td></tr>`)
	eight := strings.Index(revealed, `<tr class="vote-leader"><td>8</td><td>1</td><td>50%</td></tr>`)
	if five < 0 || eight < 0 || five > eight {
		t.Fatalf("tied counts should both be highlighted, 5 then 8: %s", revealed)
	}
	waitForMessage(t, bob, `<td class="vote-flash">Bob</td><td class="vote-flash">5</td>`)
}

func TestAdminAlwaysShowVotesAndClearVotes(t *testing.T) {
	srv := httptest.NewServer(newRouter(newApp()))
	t.Cleanup(srv.Close)

	roomID := createRoom(t, srv, "sprint")
	ada := dialRoom(t, srv, roomID, "Ada")
	waitForMessage(t, ada, `<td class="vote-flash">Ada</td><td class="vote-flash"></td>`)
	bob := dialRoom(t, srv, roomID, "Bob")
	waitForMessage(t, ada, `<td class="vote-flash">Bob</td><td class="vote-flash"></td>`)
	waitForMessage(t, bob, "<td>Ada</td><td></td>")

	if err := ada.WriteMessage(websocket.TextMessage, []byte(`{"points":"8"}`)); err != nil {
		t.Fatalf("ada vote: %v", err)
	}
	waitForMessage(t, bob, `<td class="vote-flash">Ada</td><td class="vote-flash">???</td>`)

	if err := ada.WriteMessage(websocket.TextMessage, []byte(`{"admin":"always-show-votes"}`)); err != nil {
		t.Fatalf("always show: %v", err)
	}
	shown := waitForMessage(t, bob, "<td>Ada</td><td>8</td>")
	if strings.Contains(shown, "???") {
		t.Fatalf("votes should be unmasked: %s", shown)
	}
	if !strings.Contains(shown, `aria-pressed="true"`) {
		t.Fatalf("always-show should be on: %s", shown)
	}
	if !strings.Contains(shown, `id="vote-results" class="user-table results-table" hx-swap-oob="true" hidden`) {
		t.Fatalf("always-show must not reveal the results table early: %s", shown)
	}
	waitForMessage(t, ada, "<td>Ada</td><td>8</td>")

	if err := bob.WriteMessage(websocket.TextMessage, []byte(`{"admin":"clear-votes"}`)); err != nil {
		t.Fatalf("clear votes: %v", err)
	}
	cleared := waitForMessage(t, ada, "<td>Ada</td><td></td>")
	if !strings.Contains(cleared, "<td>Bob</td><td></td>") {
		t.Fatalf("all votes should be blank: %s", cleared)
	}
	if !strings.Contains(cleared, `id="vote-results" class="user-table results-table" hx-swap-oob="true" hidden`) {
		t.Fatalf("results should hide after votes are cleared: %s", cleared)
	}
	waitForMessage(t, bob, "<td>Ada</td><td></td>")
}

func TestObserverModeClearsVoteAndIsIgnoredForMasking(t *testing.T) {
	srv := httptest.NewServer(newRouter(newApp()))
	t.Cleanup(srv.Close)

	roomID := createRoom(t, srv, "sprint")
	ada := dialRoom(t, srv, roomID, "Ada")
	waitForMessage(t, ada, `<td class="vote-flash">Ada</td><td class="vote-flash"></td>`)
	bob := dialRoom(t, srv, roomID, "Bob")
	waitForMessage(t, ada, `<td class="vote-flash">Bob</td><td class="vote-flash"></td>`)
	waitForMessage(t, bob, "<td>Ada</td><td></td>")

	if err := ada.WriteMessage(websocket.TextMessage, []byte(`{"points":"8"}`)); err != nil {
		t.Fatalf("ada vote: %v", err)
	}
	waitForMessage(t, bob, `<td class="vote-flash">Ada</td><td class="vote-flash">???</td>`)

	if err := bob.WriteMessage(websocket.TextMessage, []byte(`{"admin":"observer-mode"}`)); err != nil {
		t.Fatalf("observer: %v", err)
	}
	revealed := waitForMessage(t, ada, `<td class="vote-flash">Bob</td><td class="vote-flash">observer</td>`)
	if !strings.Contains(revealed, "<td>Ada</td><td>8</td>") {
		t.Fatalf("Ada's vote should be revealed: %s", revealed)
	}
	if strings.Contains(revealed, "???") {
		t.Fatalf("bob as observer should not keep the round masked: %s", revealed)
	}
	if strings.Contains(revealed, `id="vote-results" class="user-table results-table" hx-swap-oob="true" hidden`) {
		t.Fatalf("results should show once the only remaining voter has voted: %s", revealed)
	}
	if !strings.Contains(revealed, `<tr class="vote-leader"><td>8</td><td>1</td><td>100%</td></tr>`) {
		t.Fatalf("results should tally Ada only: %s", revealed)
	}
	waitForMessage(t, bob, `<td class="vote-flash">Bob</td><td class="vote-flash">observer</td>`)

	if err := bob.WriteMessage(websocket.TextMessage, []byte(`{"points":"5"}`)); err != nil {
		t.Fatalf("observer vote: %v", err)
	}
	if err := bob.SetReadDeadline(time.Now().Add(200 * time.Millisecond)); err != nil {
		t.Fatalf("deadline: %v", err)
	}
	_, msg, err := bob.ReadMessage()
	if err == nil && strings.Contains(string(msg), "<td>Bob</td><td>5</td>") {
		t.Fatalf("observer must not be able to vote: %s", msg)
	}

	if err := bob.WriteMessage(websocket.TextMessage, []byte(`{"admin":"observer-mode"}`)); err != nil {
		t.Fatalf("voter again: %v", err)
	}
	voterAgain := waitForMessage(t, ada, `<td class="vote-flash">Bob</td><td class="vote-flash"></td>`)
	if strings.Contains(voterAgain, "observer") {
		t.Fatalf("Bob should be a voter again: %s", voterAgain)
	}
	if !strings.Contains(voterAgain, "<td>Ada</td><td>???</td>") {
		t.Fatalf("Ada's vote should be masked once Bob is a voter again: %s", voterAgain)
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
