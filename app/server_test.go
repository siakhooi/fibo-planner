package main

import (
	"context"
	"net"
	"net/http"
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

func TestListenAddrEnv(t *testing.T) {
	t.Setenv(listenAddrEnv, "")
	if got := listenAddr(); got != defaultListenAddr {
		t.Fatalf("empty env: got %q", got)
	}
	t.Setenv(listenAddrEnv, ":9090")
	if got := listenAddr(); got != ":9090" {
		t.Fatalf("set env: got %q", got)
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
