package main

import (
	"bytes"
	"encoding/json"
	"io"
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
		`id="copy-room-url"`,
		`aria-label="Copy room link"`,
		`navigator.clipboard.writeText`,
		`id="user-list"`,
		`class="user-table"`,
		`scope="col">Points`,
		`id="points-form"`,
		`integrity="sha384-H5SrcfygHmAuTDZphMHqBJLc3FhssKjG7w/CeCpFReSfwBWDTKpkzPP8c+cLsK+V"`,
		`integrity="sha384-nIP+hMv+/j0KKPtmqpKlRK1ibiKk/4JWLfgfEC+HRGkMQUK2RMiK3/L2oU1RcJMb"`,
		`crossorigin="anonymous"`,
		`data-points="8"`,
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
		`Edit Preloaded Topic`,
		`fillPreloadedEditor`,
		`remainingPreloadedTopics`,
		`data.textContent`,
		`showModal`,
		`id="observer-mode"`,
		`id="user-name">Your name</h2>`,
		`tr.current-user td`,
		`#user-list tbody tr.current-user`,
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
		`Team Maturity(presets)`,
		`class="maturity-preset" data-percentage="100" data-max-spread="0" aria-pressed="true">full (100%, 0 spread)</button>`,
		`class="maturity-preset" data-percentage="80" data-max-spread="1" aria-pressed="false">good (80%, 1 spread)</button>`,
		`class="maturity-preset" data-percentage="50" data-max-spread="3" aria-pressed="false">relaxed (50%, 3 spreads)</button>`,
		`closest("button.maturity-preset")`,
		`scope="col">Count`,
		`scope="col">%`,
		`id="ws-status"`,
		`htmx:wsClose`,
		`JSON.stringify({ name: joinedName })`,
		`el.setAttribute("ws-connect", "/ws/" + roomID);`,
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

func TestRoomPageResponsiveLayout(t *testing.T) {
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
		`<details class="admin-panel"`,
		`<summary id="admin-heading">Administration</summary>`,
		`class="users-panel"`,
		`class="main-stack"`,
		`class="voter-toolbar"`,
		`display: contents`,
		`minmax(0, 1fr)`,
		`@media (max-width: 60rem)`,
		`syncAdminPanelOpen`,
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

	home, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("home: %v", err)
	}
	defer func() { _ = home.Body.Close() }()
	body, err := io.ReadAll(home.Body)
	if err != nil {
		t.Fatalf("read home: %v", err)
	}
	page := string(body)
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

func TestConsensusAgreementBroadcastAndAgreedPoints(t *testing.T) {
	srv := httptest.NewServer(newRouter(newApp()))
	t.Cleanup(srv.Close)

	roomID := createRoom(t, srv, "sprint")
	ada := dialRoom(t, srv, roomID, "Ada")
	waitForMessage(t, ada, `<td class="vote-flash">Ada</td><td class="vote-flash"></td>`)
	bob := dialRoom(t, srv, roomID, "Bob")
	waitForMessage(t, ada, `<td class="vote-flash">Bob</td><td class="vote-flash"></td>`)
	waitForMessage(t, bob, "<td>Ada</td><td></td>")
	cyd := dialRoom(t, srv, roomID, "Cyd")
	waitForMessage(t, ada, `<td class="vote-flash">Cyd</td><td class="vote-flash"></td>`)
	waitForMessage(t, bob, `<td class="vote-flash">Cyd</td><td class="vote-flash"></td>`)
	waitForMessage(t, cyd, "<td>Ada</td><td></td>")

	if err := ada.WriteMessage(websocket.TextMessage, []byte(`{"admin":"consensus-agreement","percentage":"75"}`)); err != nil {
		t.Fatalf("consensus: %v", err)
	}
	synced := waitForMessage(t, bob, `value="75"`)
	if !strings.Contains(synced, `id="consensus-percent"`) {
		t.Fatalf("bob should receive the consensus slider: %s", synced)
	}
	if !strings.Contains(synced, `>75</output>`) {
		t.Fatalf("bob's percentage readout should be 75: %s", synced)
	}
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
	revealed := waitForMessage(t, ada, `<td class="vote-flash">Cyd</td><td class="vote-flash">5</td>`)
	if !strings.Contains(revealed, "Agreed Points: <strong>N/A</strong>") {
		t.Fatalf("67%% should not meet 75%% consensus: %s", revealed)
	}

	if err := bob.WriteMessage(websocket.TextMessage, []byte(`{"admin":"consensus-agreement","percentage":"67"}`)); err != nil {
		t.Fatalf("lower consensus: %v", err)
	}
	stillNA := waitForMessage(t, ada, `value="67"`)
	if !strings.Contains(stillNA, "Agreed Points: <strong>N/A</strong>") {
		t.Fatalf("67%% should stay N/A while max spread is 0: %s", stillNA)
	}
	if !strings.Contains(stillNA, `X spread = 1 (require=0)`) {
		t.Fatalf("spread status should be unmet at 0: %s", stillNA)
	}

	if err := bob.WriteMessage(websocket.TextMessage, []byte(`{"admin":"consensus-agreement","max-spread":"1"}`)); err != nil {
		t.Fatalf("max spread: %v", err)
	}
	agreed := waitForMessage(t, ada, "Agreed Points: <strong>8</strong>")
	if !strings.Contains(agreed, `name="max-spread" min="0" max="6" step="1" value="1"`) {
		t.Fatalf("max spread slider should move to 1: %s", agreed)
	}
	if !strings.Contains(agreed, `class="agreed-yes"`) {
		t.Fatalf("matched consensus should be emphasized as yes: %s", agreed)
	}
	if !strings.Contains(agreed, `✓ spread = 1 (require <=1)`) {
		t.Fatalf("spread status should be met: %s", agreed)
	}
	waitForMessage(t, bob, "Agreed Points: <strong>8</strong>")
	waitForMessage(t, cyd, "Agreed Points: <strong>8</strong>")

	if err := ada.WriteMessage(websocket.TextMessage, []byte(`{"admin":"consensus-agreement","percentage":"80","max-spread":"1"}`)); err != nil {
		t.Fatalf("good preset: %v", err)
	}
	preset := waitForMessage(t, bob, `data-percentage="80" data-max-spread="1" aria-pressed="true"`)
	if !strings.Contains(preset, `name="percentage" min="50" max="100" step="1" value="80"`) {
		t.Fatalf("percentage slider should move to 80: %s", preset)
	}
	if !strings.Contains(preset, `name="max-spread" min="0" max="6" step="1" value="1"`) {
		t.Fatalf("max spread slider should move to 1: %s", preset)
	}
	if !strings.Contains(preset, `data-percentage="100" data-max-spread="0" aria-pressed="false"`) {
		t.Fatalf("full preset should no longer be selected: %s", preset)
	}
	waitForMessage(t, ada, `data-percentage="80" data-max-spread="1" aria-pressed="true"`)
	waitForMessage(t, cyd, `data-percentage="80" data-max-spread="1" aria-pressed="true"`)
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

func TestSetTopicKeepsVotes(t *testing.T) {
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
	if err := bob.WriteMessage(websocket.TextMessage, []byte(`{"points":"5"}`)); err != nil {
		t.Fatalf("bob vote: %v", err)
	}
	waitForMessage(t, ada, "<td>Ada</td><td>8</td>")

	if err := ada.WriteMessage(websocket.TextMessage, []byte(`{"admin":"set-topic","topic-title":"Next story"}`)); err != nil {
		t.Fatalf("set topic: %v", err)
	}
	updated := waitForMessage(t, ada, `<h2 id="topic-title" class="topic-title" hx-swap-oob="true">Next story</h2>`)
	if !strings.Contains(updated, "<td>Ada</td><td>8</td>") || !strings.Contains(updated, "<td>Bob</td><td>5</td>") {
		t.Fatalf("set topic should not clear votes: %s", updated)
	}
	if strings.Contains(updated, `id="vote-results" class="user-table results-table" hx-swap-oob="true" hidden`) {
		t.Fatalf("results should stay visible: %s", updated)
	}
}

func TestPreloadedTopicsBroadcastAndLoadNext(t *testing.T) {
	srv := httptest.NewServer(newRouter(newApp()))
	t.Cleanup(srv.Close)

	roomID := createRoom(t, srv, "sprint")
	ada := dialRoom(t, srv, roomID, "Ada")
	waitForMessage(t, ada, `<td class="vote-flash">Ada</td><td class="vote-flash"></td>`)
	bob := dialRoom(t, srv, roomID, "Bob")
	waitForMessage(t, ada, `<td class="vote-flash">Bob</td><td class="vote-flash"></td>`)
	waitForMessage(t, bob, "<td>Ada</td><td></td>")

	if err := ada.WriteMessage(websocket.TextMessage, []byte("{\"admin\":\"set-preloaded-topics\",\"preloaded-topics\":\"Alpha\\n\\nBeta\\n  \\nGamma\"}")); err != nil {
		t.Fatalf("set preloaded: %v", err)
	}
	listed := waitForMessage(t, bob, `>Load Next Topic [3]</button>`)
	if !strings.Contains(listed, `title="Next Topic: Alpha"`) {
		t.Fatalf("tooltip should show the first preloaded topic: %s", listed)
	}
	if !strings.Contains(listed, "<pre id=\"preloaded-topics-data\" hx-swap-oob=\"true\" hidden>Alpha\nBeta\nGamma</pre>") {
		t.Fatalf("broadcast should store remaining topics for every editor: %s", listed)
	}
	waitForMessage(t, ada, `>Load Next Topic [3]</button>`)

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
	loaded := waitForMessage(t, bob, `<h2 id="topic-title" class="topic-title" hx-swap-oob="true">Alpha</h2>`)
	if !strings.Contains(loaded, "<td>Ada</td><td></td>") || !strings.Contains(loaded, "<td>Bob</td><td></td>") {
		t.Fatalf("load next should clear votes: %s", loaded)
	}
	if !strings.Contains(loaded, `>Load Next Topic [2]</button>`) {
		t.Fatalf("count should drop after load: %s", loaded)
	}
	if !strings.Contains(loaded, `title="Next Topic: Beta"`) {
		t.Fatalf("tooltip should advance to the next topic: %s", loaded)
	}
	if strings.Contains(loaded, "Alpha\nBeta\nGamma") {
		t.Fatalf("loaded topic should be removed from the list: %s", loaded)
	}
	if !strings.Contains(loaded, "<pre id=\"preloaded-topics-data\" hx-swap-oob=\"true\" hidden>Beta\nGamma</pre>") {
		t.Fatalf("remaining topics should still be broadcast for every editor: %s", loaded)
	}
	waitForMessage(t, ada, `<h2 id="topic-title" class="topic-title" hx-swap-oob="true">Alpha</h2>`)

	if err := ada.WriteMessage(websocket.TextMessage, []byte(`{"admin":"load-next-topic"}`)); err != nil {
		t.Fatalf("load next 2: %v", err)
	}
	waitForMessage(t, bob, `title="Next Topic: Gamma"`)
	if err := ada.WriteMessage(websocket.TextMessage, []byte(`{"admin":"load-next-topic"}`)); err != nil {
		t.Fatalf("load next 3: %v", err)
	}
	empty := waitForMessage(t, bob, `<button type="submit" id="load-next-topic" hx-swap-oob="true" disabled>Load Next Topic</button>`)
	if !strings.Contains(empty, `<h2 id="topic-title" class="topic-title" hx-swap-oob="true">Gamma</h2>`) {
		t.Fatalf("last preloaded topic should become the current topic: %s", empty)
	}
	if strings.Contains(empty, `id="load-next-topic" hx-swap-oob="true" title=`) {
		t.Fatalf("empty list should not keep a next-topic tooltip: %s", empty)
	}
	if !strings.Contains(empty, `<pre id="preloaded-topics-data" hx-swap-oob="true" hidden></pre>`) {
		t.Fatalf("empty remaining list should clear the shared editor source: %s", empty)
	}
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
	if strings.Contains(voterAgain, ">observer</td>") {
		t.Fatalf("Bob should be a voter again: %s", voterAgain)
	}
	if !strings.Contains(voterAgain, "<td>Ada</td><td>???</td>") {
		t.Fatalf("Ada's vote should be masked once Bob is a voter again: %s", voterAgain)
	}
}

func TestObserverModeDuplicateNamesSyncsOnlySelf(t *testing.T) {
	srv := httptest.NewServer(newRouter(newApp()))
	t.Cleanup(srv.Close)

	roomID := createRoom(t, srv, "sprint")
	first := dialRoom(t, srv, roomID, "Alex")
	waitForMessage(t, first, `<tr class="current-user"><td class="vote-flash">Alex</td><td class="vote-flash"></td></tr>`)

	second := dialRoom(t, srv, roomID, "Alex")
	waitForMessage(t, second, `<tr class="current-user">`)
	waitForMessage(t, first, `<td class="vote-flash">Alex</td><td class="vote-flash"></td>`)

	if err := second.WriteMessage(websocket.TextMessage, []byte(`{"admin":"observer-mode"}`)); err != nil {
		t.Fatalf("observer: %v", err)
	}

	secondMsg := waitForMessage(t, second, `id="observer-mode" hx-swap-oob="true" aria-pressed="true"`)
	if !strings.Contains(secondMsg, `<tr class="current-user"><td class="vote-flash">Alex</td><td class="vote-flash">observer</td></tr>`) {
		t.Fatalf("second Alex should see itself as observer: %s", secondMsg)
	}

	firstMsg := waitForMessage(t, first, `<td class="vote-flash">Alex</td><td class="vote-flash">observer</td>`)
	if !strings.Contains(firstMsg, `id="observer-mode" hx-swap-oob="true" aria-pressed="false"`) {
		t.Fatalf("first Alex should stay a voter: %s", firstMsg)
	}
	if strings.Contains(firstMsg, `<tr class="current-user"><td class="vote-flash">Alex</td><td class="vote-flash">observer</td></tr>`) {
		t.Fatalf("first Alex must not treat the other Alex as self: %s", firstMsg)
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
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/" + roomID
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial %s: %v", name, err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	payload, err := json.Marshal(map[string]string{"name": name})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
		t.Fatalf("join %s: %v", name, err)
	}
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
	t.Setenv(wsOriginsEnv, " https://app.example.com/,http://localhost:3000 ")
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
	srv := httptest.NewServer(newRouter(newApp()))
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
	srv := httptest.NewServer(newRouter(newApp()))
	t.Cleanup(srv.Close)

	roomID := createRoom(t, srv, "sprint")
	conn := dialRoom(t, srv, roomID, "Ada")
	waitForMessage(t, conn, `<td class="vote-flash">Ada</td>`)

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

	srv := httptest.NewServer(newRouter(newApp()))
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
	srv := httptest.NewServer(newRouter(newApp()))
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
	srv := httptest.NewServer(newRouter(newApp()))
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
