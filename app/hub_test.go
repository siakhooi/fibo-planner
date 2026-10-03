package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/gorilla/websocket"
)

func TestNewRoomHubTruncatesName(t *testing.T) {
	t.Parallel()

	h := newRoomHub(strings.Repeat("é", maxDisplayNameLen+2))
	got := h.name()
	if got != strings.Repeat("é", maxDisplayNameLen) {
		t.Fatalf("got %d runes, want %d", utf8.RuneCountInString(got), maxDisplayNameLen)
	}
	if !utf8.ValidString(got) {
		t.Fatal("room name must be valid UTF-8")
	}
}

func TestHubAddTruncatesDisplayName(t *testing.T) {
	t.Parallel()

	h := newRoomHub("")
	c := &websocket.Conn{}
	h.add(c, strings.Repeat("é", maxDisplayNameLen+2))
	got := h.conns[c].name
	if got != strings.Repeat("é", maxDisplayNameLen) {
		t.Fatalf("got %d runes, want %d", utf8.RuneCountInString(got), maxDisplayNameLen)
	}
	if !utf8.ValidString(got) {
		t.Fatal("display name must be valid UTF-8")
	}
}

func TestHubSetTopicTruncatesMultibyte(t *testing.T) {
	t.Parallel()

	h := newRoomHub("")
	h.setTopic(strings.Repeat("é", maxTopicTitleLen+2))
	_, _, _, got, _, _, _ := h.pageView()
	if got != strings.Repeat("é", maxTopicTitleLen) {
		t.Fatalf("got %d runes, want %d", utf8.RuneCountInString(got), maxTopicTitleLen)
	}
	if !utf8.ValidString(got) {
		t.Fatal("topic must be valid UTF-8")
	}
}

func TestHubLoadNextTopicTruncatesMultibyte(t *testing.T) {
	t.Parallel()

	h := newRoomHub("")
	h.setPreloadedTopics([]string{strings.Repeat("é", maxTopicTitleLen+2)})
	h.loadNextTopic()
	_, _, _, got, _, _, _ := h.pageView()
	if got != strings.Repeat("é", maxTopicTitleLen) {
		t.Fatalf("got %d runes, want %d", utf8.RuneCountInString(got), maxTopicTitleLen)
	}
	if !utf8.ValidString(got) {
		t.Fatal("loaded topic must be valid UTF-8")
	}
}

func TestConnSetWriteAllDropsFailed(t *testing.T) {
	t.Parallel()

	alive := dialTestWS(t)
	dead := dialTestWS(t)
	_ = dead.Close()

	s := newConnSet()
	s.add(alive)
	s.add(dead)
	if s.count() != 2 {
		t.Fatalf("setup count=%d", s.count())
	}

	s.writeAll([]byte("ping"))
	if s.count() != 1 {
		t.Fatalf("after failed write count=%d want 1", s.count())
	}
}

func TestHubParticipantState(t *testing.T) {
	t.Parallel()

	h := newRoomHub("room")
	missing := &websocket.Conn{}
	ada := &websocket.Conn{}

	if n := h.add(ada, "  "); n != 1 || h.conns[ada].name != "Guest" {
		t.Fatalf("blank name: count=%d name=%q", n, h.conns[ada].name)
	}
	if h.setPoints(missing, "5") {
		t.Fatal("setPoints accepted an unknown connection")
	}
	if !h.setPoints(ada, "8") || h.conns[ada].points != "8" {
		t.Fatalf("setPoints points=%q", h.conns[ada].points)
	}
	if h.toggleObserver(missing) {
		t.Fatal("toggleObserver accepted an unknown connection")
	}
	if !h.toggleObserver(ada) || !h.conns[ada].observer || h.conns[ada].points != "" {
		t.Fatalf("observer=%v points=%q", h.conns[ada].observer, h.conns[ada].points)
	}
	if h.setPoints(ada, "5") {
		t.Fatal("setPoints accepted an observer")
	}
	if !h.toggleObserver(ada) || h.conns[ada].observer {
		t.Fatal("second toggle did not leave observer mode")
	}

	if !h.setPoints(ada, "3") {
		t.Fatal("setPoints after leaving observer mode")
	}
	h.clearVotes()
	if h.conns[ada].points != "" {
		t.Fatalf("clearVotes left %q", h.conns[ada].points)
	}

	h.setConsensusPercent(80)
	h.setMaxSpread(2)
	h.toggleAlwaysShowVotes()
	name, count, always, topic, consensus, spread, preloaded := h.pageView()
	if name != "room" || count != 1 || !always || topic != "" || consensus != 80 || spread != 2 || len(preloaded) != 0 {
		t.Fatalf("page name=%q count=%d always=%v topic=%q consensus=%d spread=%d preloaded=%v", name, count, always, topic, consensus, spread, preloaded)
	}
	h.setConsensusPercent(10)
	h.setMaxSpread(99)
	h.toggleAlwaysShowVotes()
	_, _, always, _, consensus, spread, _ = h.pageView()
	if always || consensus != defaultConsensusPercent || spread != defaultMaxSpread {
		t.Fatalf("normalized always=%v consensus=%d spread=%d", always, consensus, spread)
	}

	h.setTopic("keep")
	h.loadNextTopic()
	_, _, _, topic, _, _, preloaded = h.pageView()
	if topic != "keep" || len(preloaded) != 0 {
		t.Fatalf("empty queue topic=%q preloaded=%v", topic, preloaded)
	}

	if !h.setPoints(ada, "13") {
		t.Fatal("setPoints before loadNextTopic")
	}
	h.setPreloadedTopics([]string{"Login", "Logout"})
	h.loadNextTopic()
	_, _, _, topic, _, _, preloaded = h.pageView()
	if topic != "Login" || len(preloaded) != 1 || preloaded[0] != "Logout" || h.conns[ada].points != "" {
		t.Fatalf("topic=%q preloaded=%v points=%q", topic, preloaded, h.conns[ada].points)
	}
	if h.remove(ada) != 0 || h.count() != 0 {
		t.Fatalf("after remove count=%d", h.count())
	}
}

func TestBroadcastRoomStateDropsFailed(t *testing.T) {
	t.Parallel()

	alive := dialTestWS(t)
	dead := dialTestWS(t)
	_ = dead.Close()

	h := newRoomHub("")
	h.add(alive, "Ada")
	h.add(dead, "Bob")
	if h.count() != 2 {
		t.Fatalf("setup count=%d", h.count())
	}

	h.broadcastRoomState(nil)
	if h.count() != 1 {
		t.Fatalf("after failed write count=%d want 1", h.count())
	}
}

// dialTestWS upgrades a WebSocket and returns the server-side connection.
func dialTestWS(t *testing.T) *websocket.Conn {
	t.Helper()
	ready := make(chan *websocket.Conn, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		ready <- c
	}))
	t.Cleanup(srv.Close)

	u := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"
	client, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	select {
	case c := <-ready:
		t.Cleanup(func() { _ = c.Close() })
		return c
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for server websocket")
		return nil
	}
}
