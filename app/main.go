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
	"github.com/urfave/cli/v3"
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
	r.Get("/room.js", serveRoomJS)
	r.Get("/llms.txt", serveLLMSTxt)
	r.Get("/disclaimer", legalPage("disclaimer.html"))
	r.Get("/privacy", legalPage("privacy.html"))
	r.Get("/terms", legalPage("terms.html"))
	r.Get("/{roomID:[0-9]{6}}", app.roomPage)
	r.Get("/", app.home)
	return r
}

func init() {
	// --version prints commit and build date as well as the version string.
	cli.VersionPrinter = func(cmd *cli.Command) {
		if _, err := fmt.Fprint(cmd.Root().Writer, versioninfo.Format()); err != nil {
			fatal(err)
		}
	}
}

const lobbyListRoomsFlag = "lobby-list-rooms"

func newRootCommand() *cli.Command {
	return &cli.Command{
		Name:    "fibo-planner",
		Usage:   "real-time planning poker server",
		Version: versioninfo.Version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "addr",
				Aliases: []string{"a"},
				Value:   defaultListenAddr,
				Usage:   "listen address",
				Sources: cli.EnvVars(listenAddrEnv),
				Config:  cli.StringConfig{TrimSpace: true},
			},
			&cli.StringFlag{
				Name:    "ws-origins",
				Usage:   "comma-separated extra WebSocket origins",
				Sources: cli.EnvVars(wsOriginsEnv),
				Config:  cli.StringConfig{TrimSpace: true},
			},
			&cli.BoolFlag{
				Name:  lobbyListRoomsFlag,
				Usage: "list each open room on the lobby; FIBO_PLANNER_LOBBY_LIST_ROOMS=Y does the same when this flag is omitted",
			},
			&cli.StringFlag{
				Name:    "custom-html-dir",
				Usage:   "directory of optional HTML snippets and llms.txt",
				Sources: cli.EnvVars(customHTMLDirEnv),
				Config:  cli.StringConfig{TrimSpace: true},
			},
		},
		Action: serveAction,
	}
}

// applyServerFlags resolves listen address, WebSocket origins, the lobby room
// list, and the custom HTML directory. A flag that was passed wins over the
// matching environment variable.
func applyServerFlags(cmd *cli.Command) (addr string, listLobbyRooms bool, err error) {
	setAllowedWSOrigins(cmd.String("ws-origins"))
	if err = loadCustomContent(cmd.String("custom-html-dir")); err != nil {
		return "", false, err
	}
	return listenAddrFrom(cmd.String("addr")), lobbyListRoomsEnabled(cmd), nil
}

func lobbyListRoomsEnabled(cmd *cli.Command) bool {
	if cmd.IsSet(lobbyListRoomsFlag) {
		return cmd.Bool(lobbyListRoomsFlag)
	}
	return os.Getenv(lobbyListRoomsEnv) == "Y"
}

func serveAction(ctx context.Context, cmd *cli.Command) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	addr, listLobbyRooms, err := applyServerFlags(cmd)
	if err != nil {
		return err
	}
	srv := newHTTPServer(addr, newRouter(newAppConfig(listLobbyRooms)))
	log.Printf("Version: %s Commit: %s BuildDate: %s", versioninfo.Version, versioninfo.Commit, versioninfo.Date)
	log.Printf("listening on %s", listenLogURL(addr))
	return runServer(ctx, srv)
}

// fatal reports a fatal error and exits. Tests replace it.
var fatal = log.Fatal

func main() {
	if err := newRootCommand().Run(context.Background(), os.Args); err != nil {
		fatal(err)
	}
}
