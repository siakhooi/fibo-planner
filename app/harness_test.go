package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

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

func joinFlashRow(name string) string {
	return `<td class="vote-flash">` + name + `</td><td class="vote-flash"></td>`
}

func joinRoom(t *testing.T, srv *httptest.Server, roomName string, names ...string) (string, []*websocket.Conn) {
	t.Helper()
	if len(names) == 0 {
		t.Fatal("joinRoom needs at least one name")
	}
	roomID := createRoom(t, srv, roomName)
	conns := make([]*websocket.Conn, 0, len(names))
	for _, name := range names {
		conn := dialRoom(t, srv, roomID, name)
		own := waitForMessage(t, conn, joinFlashRow(name))
		self := `<tr class="current-user"><td class="vote-flash">` + name + `</td><td class="vote-flash"></td></tr>`
		if !strings.Contains(own, self) {
			t.Fatalf("%s should see their own join flash: %s", name, own)
		}
		for _, earlierName := range names[:len(conns)] {
			plain := "<td>" + earlierName + "</td><td></td>"
			if !strings.Contains(own, plain) {
				t.Fatalf("%s should see %s without a join flash: %s", name, earlierName, own)
			}
		}
		for i, earlier := range conns {
			msg := waitForMessage(t, earlier, joinFlashRow(name))
			if strings.Contains(msg, `<tr class="current-user"><td class="vote-flash">`) {
				t.Fatalf("%s should not flash when %s joins: %s", names[i], name, msg)
			}
		}
		conns = append(conns, conn)
	}
	return roomID, conns
}

func getResponse(t *testing.T, rawURL string) (status int, contentType, body string) {
	t.Helper()
	resp, err := http.Get(rawURL)
	if err != nil {
		t.Fatalf("GET %s: %v", rawURL, err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read %s: %v", rawURL, err)
	}
	return resp.StatusCode, resp.Header.Get("Content-Type"), string(b)
}

func getHTML(t *testing.T, srv *httptest.Server, path string, wantStatus int) string {
	t.Helper()
	status, _, body := getResponse(t, srv.URL+path)
	if status != wantStatus {
		t.Fatalf("GET %s: status %d, want %d", path, status, wantStatus)
	}
	return body
}
