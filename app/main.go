package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func newRouter(app *App) http.Handler {
	r := chi.NewRouter()
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
