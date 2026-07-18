package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	app := newApp()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/rooms", app.createRoom)
	r.Get("/ws", app.indexWS)
	r.Get("/ws/{roomID:[0-9]{6}}", app.roomWS)
	r.Get("/{roomID:[0-9]{6}}", app.roomPage)
	r.Get("/", app.home)

	addr := ":8080"
	log.Printf("listening on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}
