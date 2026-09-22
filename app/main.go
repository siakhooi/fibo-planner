package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func requestWithoutQueryParam(r *http.Request, key string) *http.Request {
	q := r.URL.Query()
	if _, ok := q[key]; !ok {
		return r
	}
	q.Del(key)
	cp := r.Clone(r.Context())
	u := *r.URL
	u.RawQuery = q.Encode()
	cp.URL = &u
	if u.RawQuery == "" {
		cp.RequestURI = u.Path
	} else {
		cp.RequestURI = u.Path + "?" + u.RawQuery
	}
	return cp
}

func stripJoinNameQuery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/ws") {
			r = requestWithoutQueryParam(r, "name")
		}
		next.ServeHTTP(w, r)
	})
}

func newRouter(app *App) http.Handler {
	r := chi.NewRouter()
	r.Use(stripJoinNameQuery)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/rooms", app.createRoom)
	r.Get("/ws", app.indexWS)
	r.Get("/ws/{roomID:[0-9]{6}}", app.roomWS)
	r.Get("/disclaimer", legalPage("disclaimer.html"))
	r.Get("/privacy", legalPage("privacy.html"))
	r.Get("/terms", legalPage("terms.html"))
	r.Get("/{roomID:[0-9]{6}}", app.roomPage)
	r.Get("/", app.home)
	return r
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	addr := listenAddr()
	srv := newHTTPServer(addr, newRouter(newApp()))
	log.Printf("listening on %s", listenLogURL(addr))
	if err := runServer(ctx, srv); err != nil {
		log.Fatal(err)
	}
}
