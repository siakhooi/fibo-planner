package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/siakhooi/fibo-planner/app/versioninfo"
)

func accessLogURI(path, rawQuery string) string {
	if strings.HasPrefix(path, "/ws") && rawQuery != "" {
		q, err := url.ParseQuery(rawQuery)
		if err != nil {
			rawQuery = ""
		} else {
			q.Del("name")
			rawQuery = q.Encode()
		}
	}
	if rawQuery == "" {
		return path
	}
	return path + "?" + rawQuery
}

type accessLogFormatter struct{}

type accessLogEntry struct {
	msg string
}

func (accessLogFormatter) NewLogEntry(r *http.Request) middleware.LogEntry {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return &accessLogEntry{
		msg: fmt.Sprintf(`"%s %s://%s%s %s" from %s`, r.Method, scheme, r.Host, accessLogURI(r.URL.Path, r.URL.RawQuery), r.Proto, r.RemoteAddr),
	}
}

func (e *accessLogEntry) Write(status, bytes int, _ http.Header, elapsed time.Duration, _ interface{}) {
	log.Printf("%s - %d %dB in %s", e.msg, status, bytes, elapsed)
}

func (e *accessLogEntry) Panic(v interface{}, stack []byte) {
	log.Printf("panic: %v\n%s", v, stack)
}

func newRouter(app *App) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestLogger(accessLogFormatter{}))
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

func hasVersionFlag(args []string) bool {
	for _, a := range args {
		if a == "--version" {
			return true
		}
	}
	return false
}

func main() {
	if hasVersionFlag(os.Args[1:]) {
		versioninfo.PrintBuildInfo()
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	addr := listenAddr()
	srv := newHTTPServer(addr, newRouter(newApp()))
	log.Printf("Version: %s Commit: %s BuildDate: %s", versioninfo.Version, versioninfo.Commit, versioninfo.Date)
	log.Printf("listening on %s", listenLogURL(addr))
	if err := runServer(ctx, srv); err != nil {
		log.Fatal(err)
	}
}
