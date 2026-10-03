package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
)

const (
	listenAddrEnv     = "FIBO_PLANNER_ADDR"
	defaultListenAddr = ":8080"
	readHeaderTimeout = 10 * time.Second
	idleTimeout       = 60 * time.Second
	shutdownTimeout   = 10 * time.Second
)

func listenAddrFrom(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return defaultListenAddr
	}
	return v
}

func listenLogURL(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "http://localhost" + addr // sonar-resolve [accept] go:S5332 "plain HTTP listener; TLS is terminated in front"
	}
	return "http://" + addr // sonar-resolve [accept] go:S5332 "plain HTTP listener; TLS is terminated in front"
}

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		IdleTimeout:       idleTimeout,
	}
}

func runServer(ctx context.Context, srv *http.Server) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()
	return waitServer(ctx, srv, errCh)
}

func waitServer(ctx context.Context, srv *http.Server, errCh <-chan error) error {
	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return err
		}
		err := <-errCh
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}
