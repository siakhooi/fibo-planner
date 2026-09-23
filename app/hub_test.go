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

	h := newHub()
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

	h := newHub()
	h.setTopic(strings.Repeat("é", maxTopicTitleLen+2))
	got := h.topic()
	if got != strings.Repeat("é", maxTopicTitleLen) {
		t.Fatalf("got %d runes, want %d", utf8.RuneCountInString(got), maxTopicTitleLen)
	}
	if !utf8.ValidString(got) {
		t.Fatal("topic must be valid UTF-8")
	}
}

func TestHubLoadNextTopicTruncatesMultibyte(t *testing.T) {
	t.Parallel()

	h := newHub()
	h.setPreloadedTopics([]string{strings.Repeat("é", maxTopicTitleLen+2)})
	h.loadNextTopic()
	got := h.topic()
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

func TestBroadcastRoomStateDropsFailed(t *testing.T) {
	t.Parallel()

	alive := dialTestWS(t)
	dead := dialTestWS(t)
	_ = dead.Close()

	h := newHub()
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
