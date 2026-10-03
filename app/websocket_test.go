package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestRoomPageHasPointsTable(t *testing.T) {
	srv := httptest.NewServer(newRouter(newAppConfig(false)))
	t.Cleanup(srv.Close)

	roomID := createRoom(t, srv, "sprint")
	page := getHTML(t, srv, "/"+roomID, http.StatusOK)
	for _, want := range []string{
		`id="copy-room-url"`,
		`aria-label="Copy room link"`,
		`id="user-list"`,
		`class="user-table"`,
		`scope="col">Points`,
		`id="points-form"`,
		`class="js-ws-send"`,
		`integrity="sha384-H5SrcfygHmAuTDZphMHqBJLc3FhssKjG7w/CeCpFReSfwBWDTKpkzPP8c+cLsK+V"`,
		`integrity="sha384-nIP+hMv+/j0KKPtmqpKlRK1ibiKk/4JWLfgfEC+HRGkMQUK2RMiK3/L2oU1RcJMb"`,
		`crossorigin="anonymous"`,
		`data-points="8"`,
		`aria-label="Clear vote"`,
		`Administration`,
		`aria-labelledby="admin-heading"`,
		`id="always-show-votes"`,
		`id="clear-votes"`,
		`id="set-topic"`,
		`id="topic-title"`,
		`id="topic-title-input"`,
		`id="load-next-topic"`,
		`id="edit-preloaded-topics"`,
		`id="preloaded-topics-dialog"`,
		`id="preloaded-topics-data"`,
		`Load Next Topic`,
		`Edit Preloaded Topics`,
		`src="/room.js"`,
		`data-room-id="` + roomID + `"`,
		`id="observer-mode"`,
		`I'm an observer`,
		`id="user-name">Your name</h2>`,
		`class="results-panel"`,
		`aria-labelledby="results-heading"`,
		`id="vote-results"`,
		`id="agreed-points"`,
		`Consensus Agreement`,
		`id="consensus-percent"`,
		`list="consensus-majors"`,
		`name="percentage"`,
		`min="50"`,
		`max="100"`,
		`step="1"`,
		`id="consensus-percent-value" for="consensus-percent">100</output>`,
		`id="consensus-max-spread"`,
		`name="max-spread"`,
		`id="consensus-max-spread-value" for="consensus-max-spread">0</output>`,
		`id="agreement-status"`,
		`Team maturity (presets)`,
		`class="maturity-preset" data-percentage="100" data-max-spread="0" aria-pressed="true">full (100%, 0 spread)</button>`,
		`class="maturity-preset" data-percentage="80" data-max-spread="1" aria-pressed="false">good (80%, 1 spread)</button>`,
		`class="maturity-preset" data-percentage="50" data-max-spread="3" aria-pressed="false">relaxed (50%, spread of 3)</button>`,
		`scope="col">Count`,
		`scope="col">%`,
		`id="ws-status"`,
		`Disconnected from the room. Reconnecting`,
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("room page missing %q", want)
		}
	}
	if strings.Contains(page, `?name=`) {
		t.Fatal("room WS URL should not put the display name in the query string")
	}
	if strings.Contains(page, `class="user-list"`) {
		t.Fatal("room page still has the old user-list ul")
	}
}

func TestRoomJSHasClientBehavior(t *testing.T) {
	srv := httptest.NewServer(newRouter(newAppConfig(false)))
	t.Cleanup(srv.Close)

	status, ct, js := getResponse(t, srv.URL+"/room.js")
	if status != http.StatusOK {
		t.Fatalf("room.js status %d", status)
	}
	if !strings.Contains(ct, "javascript") {
		t.Fatalf("room.js content-type %q", ct)
	}
	for _, want := range []string{
		`document.body.dataset.roomId`,
		`navigator.clipboard?.writeText`,
		`querySelectorAll(".js-ws-send")`,
		`el.setAttribute("ws-connect", "/ws/" + roomID)`,
		`htmx:wsClose`,
		`JSON.stringify({ name: joinedName })`,
		`fillPreloadedEditor`,
		`remainingPreloadedTopics`,
		`kept.length >= maxPreloadedTopics`,
		`closest("button.maturity-preset")`,
		`syncAdminPanelOpen`,
	} {
		if !strings.Contains(js, want) {
			t.Fatalf("room.js missing %q", want)
		}
	}
	if strings.Contains(js, "\nvar ") || strings.HasPrefix(js, "var ") {
		t.Fatal("room.js should use const/let, not var")
	}
}

func TestRoomPageResponsiveLayout(t *testing.T) {
	srv := httptest.NewServer(newRouter(newAppConfig(false)))
	t.Cleanup(srv.Close)

	roomID := createRoom(t, srv, "sprint")
	page := getHTML(t, srv, "/"+roomID, http.StatusOK)

	for _, want := range []string{
		`<details class="admin-panel"`,
		`<summary id="admin-heading">Administration</summary>`,
		`class="users-panel"`,
		`class="main-stack"`,
		`class="voter-toolbar"`,
		`display: contents`,
		`minmax(0, 1fr)`,
		`@media (max-width: 60rem)`,
		`src="/room.js"`,
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("room page missing %q", want)
		}
	}

	adminAt := strings.Index(page, `class="admin-panel"`)
	bodyAt := strings.Index(page, `class="room-body"`)
	resultsAt := strings.Index(page, `class="results-panel"`)
	usersAt := strings.Index(page, `class="users-panel"`)
	observerAt := strings.Index(page, `id="observer-mode"`)
	if adminAt < 0 || bodyAt < 0 || resultsAt < 0 || usersAt < 0 || observerAt < 0 {
		t.Fatal("room page missing layout landmarks")
	}
	if observerAt < bodyAt || observerAt > usersAt {
		t.Fatal("observer control should live in the main room body, not the admin panel")
	}
	if adminAt >= bodyAt || bodyAt >= usersAt || usersAt >= resultsAt {
		t.Fatal("expected DOM order admin, body, users, results")
	}
}

func TestCreateRoomWithoutName(t *testing.T) {
	srv := httptest.NewServer(newRouter(newAppConfig(false)))
	t.Cleanup(srv.Close)

	roomID := createRoom(t, srv, "")

	page := getHTML(t, srv, "/", http.StatusOK)
	if strings.Contains(page, "Room "+roomID) || strings.Contains(page, `href="/`+roomID+`"`) {
		t.Fatalf("default lobby should not list rooms: %s", page)
	}
	if !strings.Contains(page, `id="room-count">1</strong>`) {
		t.Fatalf("lobby should show room count 1: %s", page)
	}
	if !strings.Contains(page, `id="rooms-user-count">0</strong>`) {
		t.Fatalf("lobby should show 0 people in rooms: %s", page)
	}
	if !strings.Contains(page, `integrity="sha384-H5SrcfygHmAuTDZphMHqBJLc3FhssKjG7w/CeCpFReSfwBWDTKpkzPP8c+cLsK+V"`) ||
		!strings.Contains(page, `integrity="sha384-nIP+hMv+/j0KKPtmqpKlRK1ibiKk/4JWLfgfEC+HRGkMQUK2RMiK3/L2oU1RcJMb"`) ||
		!strings.Contains(page, `crossorigin="anonymous"`) {
		t.Fatalf("lobby scripts should use SRI: %s", page)
	}

	_ = getHTML(t, srv, "/"+roomID, http.StatusOK)
}

func TestVoteBroadcastToAllParticipants(t *testing.T) {
	srv := httptest.NewServer(newRouter(newAppConfig(false)))
	t.Cleanup(srv.Close)

	_, conns := joinRoom(t, srv, "sprint", "Ada", "Bob")
	ada, bob := conns[0], conns[1]

	if err := ada.WriteMessage(websocket.TextMessage, []byte(`{"points":"8"}`)); err != nil {
		t.Fatalf("ada vote: %v", err)
	}
	waitForMessage(t, ada, "???")
	waitForMessage(t, bob, "???")

	if err := bob.WriteMessage(websocket.TextMessage, []byte(`{"points":"5"}`)); err != nil {
		t.Fatalf("bob vote: %v", err)
	}
	waitForMessage(t, ada, "<td>Ada</td><td>8</td>")
	waitForMessage(t, bob, "<td>Ada</td><td>8</td>")
}

func TestConsensusAgreementBroadcastAndAgreedPoints(t *testing.T) {
	srv := httptest.NewServer(newRouter(newAppConfig(false)))
	t.Cleanup(srv.Close)

	_, conns := joinRoom(t, srv, "sprint", "Ada", "Bob", "Cyd")
	ada, bob, cyd := conns[0], conns[1], conns[2]

	if err := ada.WriteMessage(websocket.TextMessage, []byte(`{"admin":"consensus-agreement","percentage":"75"}`)); err != nil {
		t.Fatalf("consensus: %v", err)
	}
	waitForMessage(t, bob, `value="75"`)
	waitForMessage(t, ada, `value="75"`)
	waitForMessage(t, cyd, `value="75"`)

	if err := ada.WriteMessage(websocket.TextMessage, []byte(`{"points":"8"}`)); err != nil {
		t.Fatalf("ada vote: %v", err)
	}
	waitForMessage(t, bob, `<td class="vote-flash">Ada</td>`)
	if err := bob.WriteMessage(websocket.TextMessage, []byte(`{"points":"8"}`)); err != nil {
		t.Fatalf("bob vote: %v", err)
	}
	waitForMessage(t, ada, `<td class="vote-flash">Bob</td>`)
	if err := cyd.WriteMessage(websocket.TextMessage, []byte(`{"points":"5"}`)); err != nil {
		t.Fatalf("cyd vote: %v", err)
	}
	waitForMessage(t, ada, "Agreed Points: <strong>N/A</strong>")

	if err := bob.WriteMessage(websocket.TextMessage, []byte(`{"admin":"consensus-agreement","percentage":"67"}`)); err != nil {
		t.Fatalf("lower consensus: %v", err)
	}
	waitForMessage(t, ada, `value="67"`)

	if err := bob.WriteMessage(websocket.TextMessage, []byte(`{"admin":"consensus-agreement","max-spread":"1"}`)); err != nil {
		t.Fatalf("max spread: %v", err)
	}
	waitForMessage(t, ada, "Agreed Points: <strong>8</strong>")
	waitForMessage(t, bob, "Agreed Points: <strong>8</strong>")
	waitForMessage(t, cyd, "Agreed Points: <strong>8</strong>")

	if err := ada.WriteMessage(websocket.TextMessage, []byte(`{"admin":"consensus-agreement","percentage":"80","max-spread":"1"}`)); err != nil {
		t.Fatalf("good preset: %v", err)
	}
	waitForMessage(t, bob, `data-percentage="80" data-max-spread="1" aria-pressed="true"`)
	waitForMessage(t, ada, `data-percentage="80" data-max-spread="1" aria-pressed="true"`)
	waitForMessage(t, cyd, `data-percentage="80" data-max-spread="1" aria-pressed="true"`)
}

func TestAdminAlwaysShowVotesAndClearVotes(t *testing.T) {
	srv := httptest.NewServer(newRouter(newAppConfig(false)))
	t.Cleanup(srv.Close)

	_, conns := joinRoom(t, srv, "sprint", "Ada", "Bob")
	ada, bob := conns[0], conns[1]

	if err := ada.WriteMessage(websocket.TextMessage, []byte(`{"points":"8"}`)); err != nil {
		t.Fatalf("ada vote: %v", err)
	}
	waitForMessage(t, bob, `<td class="vote-flash">Ada</td><td class="vote-flash">???</td>`)

	if err := ada.WriteMessage(websocket.TextMessage, []byte(`{"admin":"always-show-votes"}`)); err != nil {
		t.Fatalf("always show: %v", err)
	}
	waitForMessage(t, bob, "<td>Ada</td><td>8</td>")
	waitForMessage(t, ada, "<td>Ada</td><td>8</td>")

	if err := bob.WriteMessage(websocket.TextMessage, []byte(`{"admin":"clear-votes"}`)); err != nil {
		t.Fatalf("clear votes: %v", err)
	}
	waitForMessage(t, ada, "<td>Ada</td><td></td>")
	waitForMessage(t, bob, "<td>Ada</td><td></td>")
}

func TestSetTopicKeepsVotes(t *testing.T) {
	srv := httptest.NewServer(newRouter(newAppConfig(false)))
	t.Cleanup(srv.Close)

	_, conns := joinRoom(t, srv, "sprint", "Ada", "Bob")
	ada, bob := conns[0], conns[1]

	if err := ada.WriteMessage(websocket.TextMessage, []byte(`{"points":"8"}`)); err != nil {
		t.Fatalf("ada vote: %v", err)
	}
	waitForMessage(t, bob, `<td class="vote-flash">Ada</td><td class="vote-flash">???</td>`)
	if err := bob.WriteMessage(websocket.TextMessage, []byte(`{"points":"5"}`)); err != nil {
		t.Fatalf("bob vote: %v", err)
	}
	waitForMessage(t, ada, "<td>Ada</td><td>8</td>")

	if err := ada.WriteMessage(websocket.TextMessage, []byte(`{"admin":"set-topic","topic-title":"Next story"}`)); err != nil {
		t.Fatalf("set topic: %v", err)
	}
	updated := waitForMessage(t, ada, "Next story")
	if !strings.Contains(updated, "<td>Ada</td><td>8</td>") || !strings.Contains(updated, "<td>Bob</td><td>5</td>") {
		t.Fatalf("set topic should not clear votes: %s", updated)
	}
}

func TestPreloadedTopicsBroadcastAndLoadNext(t *testing.T) {
	srv := httptest.NewServer(newRouter(newAppConfig(false)))
	t.Cleanup(srv.Close)

	_, conns := joinRoom(t, srv, "sprint", "Ada", "Bob")
	ada, bob := conns[0], conns[1]

	if err := ada.WriteMessage(websocket.TextMessage, []byte("{\"admin\":\"set-preloaded-topics\",\"preloaded-topics\":\"Alpha\\n\\nBeta\\n  \\nGamma\"}")); err != nil {
		t.Fatalf("set preloaded: %v", err)
	}
	waitForMessage(t, bob, "Load Next Topic [3]")
	waitForMessage(t, ada, "Load Next Topic [3]")

	if err := ada.WriteMessage(websocket.TextMessage, []byte(`{"points":"8"}`)); err != nil {
		t.Fatalf("ada vote: %v", err)
	}
	waitForMessage(t, bob, `<td class="vote-flash">Ada</td><td class="vote-flash">???</td>`)
	if err := bob.WriteMessage(websocket.TextMessage, []byte(`{"points":"5"}`)); err != nil {
		t.Fatalf("bob vote: %v", err)
	}
	waitForMessage(t, ada, "<td>Ada</td><td>8</td>")

	if err := ada.WriteMessage(websocket.TextMessage, []byte(`{"admin":"load-next-topic"}`)); err != nil {
		t.Fatalf("load next: %v", err)
	}
	loaded := waitForMessage(t, bob, ">Alpha</h2>")
	if !strings.Contains(loaded, "<td>Ada</td><td></td>") || !strings.Contains(loaded, "<td>Bob</td><td></td>") {
		t.Fatalf("load next should clear votes: %s", loaded)
	}
	waitForMessage(t, ada, ">Alpha</h2>")

	if err := ada.WriteMessage(websocket.TextMessage, []byte(`{"admin":"load-next-topic"}`)); err != nil {
		t.Fatalf("load next 2: %v", err)
	}
	waitForMessage(t, bob, ">Beta</h2>")
	if err := ada.WriteMessage(websocket.TextMessage, []byte(`{"admin":"load-next-topic"}`)); err != nil {
		t.Fatalf("load next 3: %v", err)
	}
	waitForMessage(t, bob, ">Gamma</h2>")
}

func TestObserverModeClearsVoteAndIsIgnoredForMasking(t *testing.T) {
	srv := httptest.NewServer(newRouter(newAppConfig(false)))
	t.Cleanup(srv.Close)

	_, conns := joinRoom(t, srv, "sprint", "Ada", "Bob")
	ada, bob := conns[0], conns[1]

	if err := ada.WriteMessage(websocket.TextMessage, []byte(`{"points":"8"}`)); err != nil {
		t.Fatalf("ada vote: %v", err)
	}
	waitForMessage(t, bob, `<td class="vote-flash">Ada</td><td class="vote-flash">???</td>`)

	if err := bob.WriteMessage(websocket.TextMessage, []byte(`{"admin":"observer-mode"}`)); err != nil {
		t.Fatalf("observer: %v", err)
	}
	waitForMessage(t, ada, "<td>Ada</td><td>8</td>")
	waitForMessage(t, bob, "observer")

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
	waitForMessage(t, ada, "???")
}

func TestObserverModeDuplicateNamesSyncsOnlySelf(t *testing.T) {
	srv := httptest.NewServer(newRouter(newAppConfig(false)))
	t.Cleanup(srv.Close)

	_, conns := joinRoom(t, srv, "sprint", "Alex", "Alex")
	first, second := conns[0], conns[1]

	if err := second.WriteMessage(websocket.TextMessage, []byte(`{"admin":"observer-mode"}`)); err != nil {
		t.Fatalf("observer: %v", err)
	}

	waitForMessage(t, second, `id="observer-mode" hx-swap-oob="true" aria-pressed="true"`)
	waitForMessage(t, first, `id="observer-mode" hx-swap-oob="true" aria-pressed="false"`)
}

func TestCheckWSOrigin(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://planner.example/ws", nil)
	req.Host = "planner.example"
	if !checkWSOrigin(req) {
		t.Fatal("missing Origin should be allowed")
	}
	req.Header.Set("Origin", "http://planner.example")
	if !checkWSOrigin(req) {
		t.Fatal("same origin should be allowed")
	}
	req.Header.Set("Origin", "https://evil.example")
	if checkWSOrigin(req) {
		t.Fatal("cross origin should be rejected")
	}
	req.Header.Set("Origin", "://bad")
	if checkWSOrigin(req) {
		t.Fatal("invalid origin should be rejected")
	}
}

func TestCheckWSOriginAllowlist(t *testing.T) {
	setAllowedWSOrigins(" https://app.example.com/,http://localhost:3000 ")
	t.Cleanup(func() { setAllowedWSOrigins("") })
	req := httptest.NewRequest(http.MethodGet, "http://internal:8080/ws", nil)
	req.Host = "internal:8080"
	req.Header.Set("Origin", "https://app.example.com")
	if !checkWSOrigin(req) {
		t.Fatal("allowlisted origin should be allowed")
	}
	req.Header.Set("Origin", "https://other.example")
	if checkWSOrigin(req) {
		t.Fatal("non-allowlisted origin should be rejected")
	}
}

func TestParseWSOrigins(t *testing.T) {
	got := parseWSOrigins(" https://a.example/, ,https://b.example ")
	if len(got) != 2 || got[0] != "https://a.example" || got[1] != "https://b.example" {
		t.Fatalf("got %#v", got)
	}
	if parseWSOrigins("  ") != nil && len(parseWSOrigins("  ")) != 0 {
		t.Fatalf("blank should be empty")
	}
}

func TestWSUpgradeRejectsCrossOrigin(t *testing.T) {
	srv := httptest.NewServer(newRouter(newAppConfig(false)))
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}

func TestWSOversizedMessageCloses(t *testing.T) {
	srv := httptest.NewServer(newRouter(newAppConfig(false)))
	t.Cleanup(srv.Close)

	_, conns := joinRoom(t, srv, "sprint", "Ada")
	conn := conns[0]

	payload := bytes.Repeat([]byte("a"), maxWSMessageBytes+8)
	if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
		t.Fatalf("write huge: %v", err)
	}
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("expected close after oversized message")
	}
}

func TestWSPingKeepsConnection(t *testing.T) {
	origWait, origPeriod := wsPongWait, wsPingPeriod
	t.Cleanup(func() {
		wsPongWait = origWait
		wsPingPeriod = origPeriod
	})
	wsPongWait = time.Second
	wsPingPeriod = 200 * time.Millisecond

	srv := httptest.NewServer(newRouter(newAppConfig(false)))
	t.Cleanup(srv.Close)

	roomID := createRoom(t, srv, "sprint")
	conn := dialRoom(t, srv, roomID, "Ada")

	var writeMu sync.Mutex
	pings := make(chan struct{}, 8)
	conn.SetPingHandler(func(appData string) error {
		select {
		case pings <- struct{}{}:
		default:
		}
		writeMu.Lock()
		defer writeMu.Unlock()
		return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(time.Second))
	})

	msgs := make(chan string, 16)
	readErr := make(chan error, 1)
	go func() {
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				readErr <- err
				return
			}
			msgs <- string(data)
		}
	}()

	waitMsg := func(substr string) {
		t.Helper()
		timeout := time.After(2 * time.Second)
		var last string
		for {
			select {
			case m := <-msgs:
				last = m
				if strings.Contains(m, substr) {
					return
				}
			case err := <-readErr:
				t.Fatalf("ws read waiting for %q: %v (last=%q)", substr, err, last)
			case <-timeout:
				t.Fatalf("timeout waiting for %q (last=%q)", substr, last)
			}
		}
	}

	waitMsg(`<td class="vote-flash">Ada</td>`)

	select {
	case <-pings:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not send a ping")
	}

	select {
	case err := <-readErr:
		t.Fatalf("connection died during ping window: %v", err)
	case <-time.After(1100 * time.Millisecond):
	}

	writeMu.Lock()
	err := conn.WriteMessage(websocket.TextMessage, []byte(`{"points":"8"}`))
	writeMu.Unlock()
	if err != nil {
		t.Fatalf("vote after idle: %v", err)
	}
	waitMsg(`<td class="vote-flash">Ada</td><td class="vote-flash">8</td>`)
}

func TestRoomWSIgnoresNameQueryString(t *testing.T) {
	srv := httptest.NewServer(newRouter(newAppConfig(false)))
	t.Cleanup(srv.Close)

	roomID := createRoom(t, srv, "sprint")
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/" + roomID + "?name=" + url.QueryEscape("FromQuery")
	sneaky, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sneaky.Close() })

	ada := dialRoom(t, srv, roomID, "Ada")
	msg := waitForMessage(t, ada, `<td class="vote-flash">Ada</td>`)
	if strings.Contains(msg, "FromQuery") {
		t.Fatal("display name from query string should not join the room")
	}
}

func TestRoomWSVoteBeforeJoinIgnored(t *testing.T) {
	srv := httptest.NewServer(newRouter(newAppConfig(false)))
	t.Cleanup(srv.Close)

	roomID := createRoom(t, srv, "sprint")
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/" + roomID
	early, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = early.Close() })
	if err := early.WriteMessage(websocket.TextMessage, []byte(`{"points":"8"}`)); err != nil {
		t.Fatal(err)
	}

	ada := dialRoom(t, srv, roomID, "Ada")
	msg := waitForMessage(t, ada, `<td class="vote-flash">Ada</td>`)
	if strings.Contains(msg, `>8<`) || strings.Contains(msg, "Guest") {
		t.Fatalf("vote before join should be ignored, got %s", msg)
	}
}

func TestIndexWSBroadcastsOnConnect(t *testing.T) {
	a := newAppConfig(false)
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

func TestRoomWSUpgradeRejectsCrossOrigin(t *testing.T) {
	srv := httptest.NewServer(newRouter(newAppConfig(false)))
	t.Cleanup(srv.Close)

	roomID := createRoom(t, srv, "sprint")
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/ws/"+roomID, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
}

func TestRoomWSInvalidJSONIsIgnored(t *testing.T) {
	srv := httptest.NewServer(newRouter(newAppConfig(false)))
	t.Cleanup(srv.Close)

	_, conns := joinRoom(t, srv, "sprint", "Ada")
	ada := conns[0]
	if err := ada.WriteMessage(websocket.TextMessage, []byte("not-json")); err != nil {
		t.Fatal(err)
	}
	if err := ada.WriteMessage(websocket.TextMessage, []byte(`{"points":"8"}`)); err != nil {
		t.Fatal(err)
	}
	waitForMessage(t, ada, `<td class="vote-flash">Ada</td><td class="vote-flash">8</td>`)
}

func TestRoomWSInvalidVoteIsIgnored(t *testing.T) {
	srv := httptest.NewServer(newRouter(newAppConfig(false)))
	t.Cleanup(srv.Close)

	_, conns := joinRoom(t, srv, "sprint", "Ada")
	ada := conns[0]
	if err := ada.WriteMessage(websocket.TextMessage, []byte(`{"points":"99"}`)); err != nil {
		t.Fatal(err)
	}
	if err := ada.WriteMessage(websocket.TextMessage, []byte(`{"points":"5"}`)); err != nil {
		t.Fatal(err)
	}
	msg := waitForMessage(t, ada, `<td class="vote-flash">Ada</td><td class="vote-flash">5</td>`)
	if strings.Contains(msg, ">99<") {
		t.Fatalf("invalid vote should be ignored, got %s", msg)
	}
}

func TestWSPingWriteFailureStops(t *testing.T) {
	orig := wsPingPeriod
	t.Cleanup(func() { wsPingPeriod = orig })
	wsPingPeriod = 20 * time.Millisecond

	conn := dialTestWS(t)
	_ = conn.Close()
	var mu sync.Mutex
	stop := startWSPing(&mu, conn)
	t.Cleanup(stop)
	time.Sleep(100 * time.Millisecond)
}
