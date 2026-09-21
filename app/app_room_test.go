package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRoomPageNotFound(t *testing.T) {
	srv := httptest.NewServer(newRouter(newApp()))
	t.Cleanup(srv.Close)

	page := getHTML(t, srv, "/000000", http.StatusNotFound)
	if !strings.Contains(page, "Room does not exist") {
		t.Fatalf("missing not-found copy: %s", page)
	}
	if !strings.Contains(page, "<strong>000000</strong>") {
		t.Fatalf("missing room id: %s", page)
	}
}

func TestRoomPageMissingNameEntryIsNotFound(t *testing.T) {
	a := newAppConfig(false)
	a.roomHubs["123456"] = newHub()

	srv := httptest.NewServer(newRouter(a))
	t.Cleanup(srv.Close)

	page := getHTML(t, srv, "/123456", http.StatusNotFound)
	if !strings.Contains(page, "Room does not exist") {
		t.Fatalf("inconsistent maps should 404, got: %s", page)
	}
}

func TestRoomPageAfterEvictionIsNotFound(t *testing.T) {
	a := newAppConfig(false)
	h := newHub()
	a.roomHubs["123456"] = h
	a.rooms["123456"] = newRoom("sprint")
	a.evictRoomIfStillEmpty("123456", h)

	srv := httptest.NewServer(newRouter(a))
	t.Cleanup(srv.Close)

	page := getHTML(t, srv, "/123456", http.StatusNotFound)
	if !strings.Contains(page, "Room does not exist") {
		t.Fatalf("evicted room should 404: %s", page)
	}
}

func TestRoomPageNamedRoom(t *testing.T) {
	srv := httptest.NewServer(newRouter(newApp()))
	t.Cleanup(srv.Close)

	id := createRoom(t, srv, "sprint")
	page := getHTML(t, srv, "/"+id, http.StatusOK)
	if !strings.Contains(page, "<h1>Room sprint</h1>") {
		t.Fatalf("named room heading missing: %s", page)
	}
}

func TestRoomWSMissingNameEntryIsNotFound(t *testing.T) {
	a := newAppConfig(false)
	a.roomHubs["123456"] = newHub()

	srv := httptest.NewServer(newRouter(a))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/ws/123456?name=Ada")
	if err != nil {
		t.Fatalf("ws get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}
