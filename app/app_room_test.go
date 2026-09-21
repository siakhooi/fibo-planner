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

func TestRoomPageUnnamedHubUsesRoomID(t *testing.T) {
	a := newAppConfig(false)
	a.roomHubs["123456"] = newHub()

	srv := httptest.NewServer(newRouter(a))
	t.Cleanup(srv.Close)

	page := getHTML(t, srv, "/123456", http.StatusOK)
	if !strings.Contains(page, "<h1>Room 123456</h1>") {
		t.Fatalf("unnamed room should use the id: %s", page)
	}
}

func TestRoomPageAfterEvictionIsNotFound(t *testing.T) {
	a := newAppConfig(false)
	h := newRoomHub("sprint")
	a.roomHubs["123456"] = h
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
