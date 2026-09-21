package main

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
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

func TestCreateRoomTruncatesMultibyteName(t *testing.T) {
	srv := httptest.NewServer(newRouter(newApp()))
	t.Cleanup(srv.Close)

	id := createRoom(t, srv, strings.Repeat("é", maxDisplayNameLen+3))
	page := getHTML(t, srv, "/"+id, http.StatusOK)
	want := "<h1>Room " + strings.Repeat("é", maxDisplayNameLen) + "</h1>"
	if !strings.Contains(page, want) {
		t.Fatalf("room name should be %d runes: %s", maxDisplayNameLen, page)
	}
}

type failReader struct{}

func (failReader) Read([]byte) (int, error) {
	return 0, errors.New("rand unavailable")
}

func TestRandomSixDigitRoomIDFormat(t *testing.T) {
	for range 20 {
		id, err := randomSixDigitRoomID()
		if err != nil {
			t.Fatal(err)
		}
		if len(id) != 6 {
			t.Fatalf("len=%d id=%q", len(id), id)
		}
		n, err := strconv.Atoi(id)
		if err != nil {
			t.Fatal(err)
		}
		if n < 100000 || n > 999999 {
			t.Fatalf("id %s out of range", id)
		}
	}
}

func TestRandomSixDigitRoomIDError(t *testing.T) {
	orig := cryptoReader
	t.Cleanup(func() { cryptoReader = orig })
	cryptoReader = failReader{}

	id, err := randomSixDigitRoomID()
	if err == nil {
		t.Fatal("expected error")
	}
	if id != "" {
		t.Fatalf("id=%q, want empty", id)
	}
}

func TestCreateRoomCryptoFailureIs503(t *testing.T) {
	orig := cryptoReader
	t.Cleanup(func() { cryptoReader = orig })
	cryptoReader = failReader{}

	a := newAppConfig(false)
	srv := httptest.NewServer(newRouter(a))
	t.Cleanup(srv.Close)

	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.PostForm(srv.URL+"/rooms", url.Values{"name": {"sprint"}})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status %d body %s", resp.StatusCode, body)
	}
	if len(a.roomHubs) != 0 {
		t.Fatalf("must not insert a room, have %d", len(a.roomHubs))
	}
}
