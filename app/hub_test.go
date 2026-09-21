package main

import (
	"strings"
	"testing"
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
