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
	"time"

	"github.com/gorilla/websocket"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	a := newAppConfig(false)
	t.Cleanup(func() { stopRoomEvictions(a) })
	return a
}

func newRoomServer(t *testing.T, a *App) *httptest.Server {
	t.Helper()
	if a == nil {
		a = newTestApp(t)
	}
	srv := httptest.NewServer(newRouter(a))
	t.Cleanup(srv.Close)
	return srv
}

func useCryptoReader(t *testing.T, r io.Reader) {
	t.Helper()
	orig := cryptoReader
	t.Cleanup(func() { cryptoReader = orig })
	cryptoReader = r
}

func postRooms(t *testing.T, srv *httptest.Server, body string) (status int, respBody, location string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/rooms", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, string(b), resp.Header.Get("Location")
}

func stopRoomEvictions(a *App) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for id, timer := range a.roomEvictTimers {
		timer.Stop()
		delete(a.roomEvictTimers, id)
	}
}

func roomEvictionTimer(t *testing.T, a *App, roomID string) *time.Timer {
	t.Helper()
	a.mu.Lock()
	defer a.mu.Unlock()
	timer, ok := a.roomEvictTimers[roomID]
	if !ok {
		t.Fatalf("room %s has no eviction timer", roomID)
	}
	return timer
}

func triggerRoomEviction(t *testing.T, a *App, roomID string) {
	t.Helper()
	roomEvictionTimer(t, a, roomID).Reset(0)
}

func waitRoomGone(t *testing.T, a *App, roomID string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		a.mu.Lock()
		_, hubOK := a.roomHubs[roomID]
		_, timerOK := a.roomEvictTimers[roomID]
		a.mu.Unlock()
		if !hubOK && !timerOK {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("room %s was not evicted", roomID)
}

type failReader struct{}

func (failReader) Read([]byte) (int, error) {
	return 0, errors.New("rand unavailable")
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}

type byteReader struct {
	b []byte
	i int
}

func (r *byteReader) Read(p []byte) (int, error) {
	if r.i >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.i:])
	r.i += n
	return n, nil
}

func TestScheduleRoomEvictionReplacesTimer(t *testing.T) {
	a := newTestApp(t)
	h := newRoomHub("sprint")
	const id = "123456"
	a.roomHubs[id] = h
	a.scheduleRoomEviction(id, h)
	first := roomEvictionTimer(t, a, id)

	a.scheduleRoomEviction(id, h)
	second := roomEvictionTimer(t, a, id)
	if first == second {
		t.Fatal("expected a replacement idle timer")
	}
	if first.Stop() {
		t.Fatal("replaced timer should already be stopped")
	}
	if _, ok := a.getHub(id); !ok {
		t.Fatal("replacing the timer should keep the room")
	}
}

func TestCancelRoomEvictionStopsTimer(t *testing.T) {
	a := newTestApp(t)
	h := newRoomHub("sprint")
	const id = "123456"
	a.roomHubs[id] = h
	a.scheduleRoomEviction(id, h)
	timer := roomEvictionTimer(t, a, id)

	a.cancelRoomEviction(id)

	a.mu.Lock()
	_, ok := a.roomEvictTimers[id]
	a.mu.Unlock()
	if ok {
		t.Fatal("cancelled timer should be removed")
	}
	if timer.Stop() {
		t.Fatal("cancelled timer should already be stopped")
	}
	if _, still := a.getHub(id); !still {
		t.Fatal("cancelling eviction should keep the room")
	}

	a.cancelRoomEviction(id)
}

func TestIdleTimerRemovesEmptyRoom(t *testing.T) {
	a := newTestApp(t)
	h := newRoomHub("sprint")
	const id = "123456"
	a.roomHubs[id] = h
	a.scheduleRoomEviction(id, h)

	triggerRoomEviction(t, a, id)
	waitRoomGone(t, a, id)
}

func TestEvictRoomIgnoresMissingHub(t *testing.T) {
	a := newTestApp(t)
	a.evictRoomIfStillEmpty("123456", newRoomHub("sprint"))
	if len(a.roomHubs) != 0 {
		t.Fatalf("missing room should stay missing, have %d", len(a.roomHubs))
	}
}

func TestEvictRoomIgnoresReplacedHub(t *testing.T) {
	a := newTestApp(t)
	const id = "123456"
	next := newRoomHub("next")
	a.roomHubs[id] = next
	a.roomEvictTimers[id] = time.AfterFunc(time.Hour, func() {})

	a.evictRoomIfStillEmpty(id, newRoomHub("old"))

	if got, ok := a.getHub(id); !ok || got != next {
		t.Fatal("replacement hub should stay")
	}
	roomEvictionTimer(t, a, id)
}

func TestEvictRoomKeepsOccupiedRoom(t *testing.T) {
	a := newTestApp(t)
	h := newRoomHub("sprint")
	h.add(&websocket.Conn{}, "Ada")
	const id = "123456"
	a.roomHubs[id] = h
	a.scheduleRoomEviction(id, h)

	a.evictRoomIfStillEmpty(id, h)

	if got, ok := a.getHub(id); !ok || got != h {
		t.Fatal("occupied room should stay")
	}
	a.mu.Lock()
	_, timerOK := a.roomEvictTimers[id]
	a.mu.Unlock()
	if timerOK {
		t.Fatal("eviction timer should be cleared while the room is occupied")
	}
}

func TestRoomPageNotFound(t *testing.T) {
	srv := newRoomServer(t, nil)

	page := getHTML(t, srv, "/000000", http.StatusNotFound)
	if !strings.Contains(page, "Room does not exist") {
		t.Fatalf("missing not-found copy: %s", page)
	}
	if !strings.Contains(page, "<strong>000000</strong>") {
		t.Fatalf("missing room id: %s", page)
	}
}

func TestRoomPageUnnamedHubUsesRoomID(t *testing.T) {
	a := newTestApp(t)
	a.roomHubs["123456"] = newRoomHub("")
	srv := newRoomServer(t, a)

	page := getHTML(t, srv, "/123456", http.StatusOK)
	if !strings.Contains(page, "<h1>Room 123456</h1>") {
		t.Fatalf("unnamed room should use the id: %s", page)
	}
}

func TestRoomPageAfterEvictionIsNotFound(t *testing.T) {
	a := newTestApp(t)
	h := newRoomHub("sprint")
	a.roomHubs["123456"] = h
	a.evictRoomIfStillEmpty("123456", h)
	srv := newRoomServer(t, a)

	page := getHTML(t, srv, "/123456", http.StatusNotFound)
	if !strings.Contains(page, "Room does not exist") {
		t.Fatalf("evicted room should 404: %s", page)
	}
}

func TestRoomPageNamedRoom(t *testing.T) {
	srv := newRoomServer(t, nil)

	id := createRoom(t, srv, "sprint")
	page := getHTML(t, srv, "/"+id, http.StatusOK)
	if !strings.Contains(page, "<h1>Room sprint</h1>") {
		t.Fatalf("named room heading missing: %s", page)
	}
}

func TestCreateRoomTrimsName(t *testing.T) {
	srv := newRoomServer(t, nil)

	id := createRoom(t, srv, "  sprint  ")
	page := getHTML(t, srv, "/"+id, http.StatusOK)
	if !strings.Contains(page, "<h1>Room sprint</h1>") {
		t.Fatalf("room name should be trimmed: %s", page)
	}
}

func TestCreateRoomTruncatesMultibyteName(t *testing.T) {
	srv := newRoomServer(t, nil)

	id := createRoom(t, srv, strings.Repeat("é", maxDisplayNameLen+3))
	page := getHTML(t, srv, "/"+id, http.StatusOK)
	want := "<h1>Room " + strings.Repeat("é", maxDisplayNameLen) + "</h1>"
	if !strings.Contains(page, want) {
		t.Fatalf("room name should be %d runes: %s", maxDisplayNameLen, page)
	}
}

func TestCreateRoomBadFormIs400(t *testing.T) {
	a := newTestApp(t)
	srv := newRoomServer(t, a)

	status, body, _ := postRooms(t, srv, "name=%zz")
	if status != http.StatusBadRequest {
		t.Fatalf("status %d body %s", status, body)
	}
	if !strings.Contains(body, "bad form") {
		t.Fatalf("body %s", body)
	}
	if len(a.roomHubs) != 0 {
		t.Fatalf("bad form must not insert a room, have %d", len(a.roomHubs))
	}
}

func TestCreateRoomCryptoFailureIs503(t *testing.T) {
	useCryptoReader(t, failReader{})
	a := newTestApp(t)
	srv := newRoomServer(t, a)

	status, body, location := postRooms(t, srv, url.Values{"name": {"sprint"}}.Encode())
	if status != http.StatusServiceUnavailable {
		t.Fatalf("status %d body %s", status, body)
	}
	if location != "" {
		t.Fatalf("location %q", location)
	}
	if !strings.Contains(body, "could not allocate room") {
		t.Fatalf("body %s", body)
	}
	if len(a.roomHubs) != 0 {
		t.Fatalf("must not insert a room, have %d", len(a.roomHubs))
	}
}

func TestCreateRoomRetriesTakenID(t *testing.T) {
	// rand.Int reads 3 bytes for a value in [0, 900000). 0 -> 100000, 1 -> 100001.
	useCryptoReader(t, &byteReader{b: []byte{0, 0, 0, 0, 0, 1}})
	a := newTestApp(t)
	a.roomHubs["100000"] = newRoomHub("taken")
	srv := newRoomServer(t, a)

	status, body, location := postRooms(t, srv, url.Values{"name": {"sprint"}}.Encode())
	if status != http.StatusSeeOther {
		t.Fatalf("status %d body %s", status, body)
	}
	if location != "/100001" {
		t.Fatalf("location %q", location)
	}
	if _, ok := a.getHub("100000"); !ok {
		t.Fatal("taken room should stay")
	}
	h, ok := a.getHub("100001")
	if !ok || h.name() != "sprint" {
		t.Fatal("retried id should be the new room")
	}
}

func TestCreateRoomIDExhaustionIs503(t *testing.T) {
	useCryptoReader(t, zeroReader{})
	a := newTestApp(t)
	a.roomHubs["100000"] = newRoomHub("taken")
	srv := newRoomServer(t, a)

	status, body, location := postRooms(t, srv, url.Values{"name": {"sprint"}}.Encode())
	if status != http.StatusServiceUnavailable {
		t.Fatalf("status %d body %s", status, body)
	}
	if location != "" {
		t.Fatalf("location %q", location)
	}
	if !strings.Contains(body, "could not allocate room") {
		t.Fatalf("body %s", body)
	}
	a.mu.Lock()
	n := len(a.roomHubs)
	_, onlyTaken := a.roomHubs["100000"]
	a.mu.Unlock()
	if n != 1 || !onlyTaken {
		t.Fatalf("exhaustion should leave only the taken room, rooms=%d taken=%v", n, onlyTaken)
	}
}

func TestRoomWSMissingRoom(t *testing.T) {
	srv := newRoomServer(t, nil)

	status, contentType, body := getResponse(t, srv.URL+"/ws/000000")
	if status != http.StatusNotFound {
		t.Fatalf("status %d body %s", status, body)
	}
	if !strings.Contains(body, "room not found") {
		t.Fatalf("body %s", body)
	}
	if strings.Contains(contentType, "text/html") {
		t.Fatalf("missing room websocket should not render HTML, content-type %q", contentType)
	}
}

func TestRoomWSUpgradesAndCancelsEviction(t *testing.T) {
	a := newTestApp(t)
	const id = "123456"
	a.roomHubs[id] = newRoomHub("sprint")
	a.scheduleRoomEviction(id, a.roomHubs[id])
	srv := newRoomServer(t, a)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/" + id
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}

	a.mu.Lock()
	_, ok := a.roomEvictTimers[id]
	a.mu.Unlock()
	if ok {
		_ = conn.Close()
		t.Fatal("websocket connect should cancel idle eviction")
	}
	if _, ok := a.getHub(id); !ok {
		_ = conn.Close()
		t.Fatal("room should remain after connect")
	}

	// Close and wait until the handler schedules eviction again, so its
	// goroutines are finished before later tests touch shared globals.
	_ = conn.Close()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		a.mu.Lock()
		_, scheduled := a.roomEvictTimers[id]
		a.mu.Unlock()
		if scheduled {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("closing the socket should schedule idle eviction again")
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
	useCryptoReader(t, failReader{})

	id, err := randomSixDigitRoomID()
	if err == nil {
		t.Fatal("expected error")
	}
	if id != "" {
		t.Fatalf("id=%q, want empty", id)
	}
}
