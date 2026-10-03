package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestListenAddrFrom(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in, want string
	}{
		{in: "", want: defaultListenAddr},
		{in: "   ", want: defaultListenAddr},
		{in: ":9090", want: ":9090"},
		{in: " 127.0.0.1:3000 ", want: "127.0.0.1:3000"},
	}
	for _, tc := range cases {
		if got := listenAddrFrom(tc.in); got != tc.want {
			t.Errorf("listenAddrFrom(%q)=%q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestListenLogURL(t *testing.T) {
	t.Parallel()

	if got := listenLogURL(":8080"); got != "http://localhost:8080" {
		t.Fatalf("got %q", got)
	}
	if got := listenLogURL("127.0.0.1:9090"); got != "http://127.0.0.1:9090" {
		t.Fatalf("got %q", got)
	}
}

func TestNewHTTPServerTimeouts(t *testing.T) {
	t.Parallel()

	h := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	srv := newHTTPServer(":9090", h)
	if srv.Addr != ":9090" {
		t.Fatalf("Addr=%q", srv.Addr)
	}
	if srv.Handler == nil {
		t.Fatal("missing handler")
	}
	if srv.ReadHeaderTimeout != readHeaderTimeout {
		t.Fatalf("ReadHeaderTimeout=%v", srv.ReadHeaderTimeout)
	}
	if srv.IdleTimeout != idleTimeout {
		t.Fatalf("IdleTimeout=%v", srv.IdleTimeout)
	}
	if srv.ReadTimeout != 0 {
		t.Fatalf("ReadTimeout=%v; must stay 0 so WebSockets are not killed", srv.ReadTimeout)
	}
	if srv.WriteTimeout != 0 {
		t.Fatalf("WriteTimeout=%v; must stay 0 so WebSockets are not killed", srv.WriteTimeout)
	}
}

func TestWaitServerShutdown(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	srv := newHTTPServer(ln.Addr().String(), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ln)
	}()

	resp, err := http.Get("http://" + ln.Addr().String() + "/")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d", resp.StatusCode)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan error, 1)
	go func() {
		done <- waitServer(ctx, srv, errCh)
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown timed out")
	}
}

func TestRunServerReportsListenError(t *testing.T) {
	srv := newHTTPServer("127.0.0.1:99999", http.NewServeMux())
	err := runServer(context.Background(), srv)
	if err == nil {
		t.Fatal("expected listen error")
	}
	if errors.Is(err, http.ErrServerClosed) {
		t.Fatalf("listen error = %v", err)
	}
}

func TestWaitServerIgnoresErrServerClosed(t *testing.T) {
	errCh := make(chan error, 1)
	errCh <- http.ErrServerClosed
	if err := waitServer(context.Background(), newHTTPServer("127.0.0.1:0", http.NewServeMux()), errCh); err != nil {
		t.Fatal(err)
	}
}

func TestWaitServerReturnsServeErrorAfterShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// Leave errCh empty so the cancelled context wins the select. Shutdown of a
	// server that never listened returns immediately, then this error is read.
	errCh := make(chan error)
	go func() {
		time.Sleep(50 * time.Millisecond)
		errCh <- errors.New("serve failed")
	}()
	err := waitServer(ctx, newHTTPServer("127.0.0.1:0", http.NewServeMux()), errCh)
	if err == nil || err.Error() != "serve failed" {
		t.Fatalf("got %v", err)
	}
}

func TestWaitServerShutdownTimesOut(t *testing.T) {
	orig := shutdownTimeout
	t.Cleanup(func() { shutdownTimeout = orig })
	shutdownTimeout = 30 * time.Millisecond

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	hold := make(chan struct{})
	var once sync.Once
	release := func() { once.Do(func() { close(hold) }) }
	t.Cleanup(release)

	srv := newHTTPServer(ln.Addr().String(), http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		close(started)
		<-hold
	}))
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ln)
	}()
	t.Cleanup(func() { _ = srv.Close() })

	var conn net.Conn
	deadline := time.Now().Add(2 * time.Second)
	for {
		c, dialErr := net.Dial("tcp", ln.Addr().String())
		if dialErr == nil {
			conn = c
			break
		}
		if time.Now().After(deadline) {
			t.Fatal(dialErr)
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if _, err := conn.Write([]byte("GET / HTTP/1.1\r\nHost: localhost\r\n\r\n")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not start")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := waitServer(ctx, srv, errCh); err == nil {
		t.Fatal("expected shutdown to time out while a request is in flight")
	}
}

func TestAccessLogURIRedactsWSName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		path, query, want string
	}{
		{path: "/ws/123456", query: "name=Ada&keep=1", want: "/ws/123456?keep=1"},
		{path: "/ws/123456", query: "name=Ada", want: "/ws/123456"},
		{path: "/ws", query: "name=Ada", want: "/ws"},
		{path: "/rooms", query: "name=sprint", want: "/rooms?name=sprint"},
		{path: "/123456", query: "", want: "/123456"},
	}
	for _, tc := range cases {
		if got := accessLogURI(tc.path, tc.query); got != tc.want {
			t.Errorf("accessLogURI(%q, %q)=%q, want %q", tc.path, tc.query, got, tc.want)
		}
	}
}
