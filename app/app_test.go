package main

import "testing"

func TestNewAppConfig(t *testing.T) {
	t.Parallel()

	for _, list := range []bool{false, true} {
		a := newAppConfig(list)
		if a.indexConns == nil || a.roomHubs == nil || a.roomEvictTimers == nil || a.listLobbyRooms != list {
			t.Fatalf("list=%v index=%v hubs=%v timers=%v flag=%v", list, a.indexConns != nil, a.roomHubs != nil, a.roomEvictTimers != nil, a.listLobbyRooms)
		}
		if a.indexConns.count() != 0 || len(a.roomHubs) != 0 || len(a.roomEvictTimers) != 0 {
			t.Fatal("expected an empty app")
		}
	}
}

func TestGetHub(t *testing.T) {
	t.Parallel()

	a := newAppConfig(false)
	if _, ok := a.getHub("100000"); ok {
		t.Fatal("missing room was found")
	}

	h := newRoomHub("Ada")
	a.mu.Lock()
	a.roomHubs["100000"] = h
	a.mu.Unlock()

	got, ok := a.getHub("100000")
	if !ok || got != h {
		t.Fatalf("got %p ok=%v", got, ok)
	}
	if _, ok := a.getHub("100001"); ok {
		t.Fatal("other room was found")
	}
}
