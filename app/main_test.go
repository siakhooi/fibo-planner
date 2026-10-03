package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/siakhooi/fibo-planner/app/versioninfo"
	"github.com/urfave/cli/v3"
)

func TestRootCommandVersion(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{
		{"fibo-planner", "--version"},
		{"fibo-planner", "-v"},
	} {
		t.Run(args[1], func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			cmd := newRootCommand()
			cmd.Writer = &out
			if err := cmd.Run(context.Background(), args); err != nil {
				t.Fatal(err)
			}
			if got := out.String(); got != versioninfo.Format() {
				t.Fatalf("version output %q", got)
			}
		})
	}
}

func TestRootCommandHelp(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{
		{"fibo-planner", "--help"},
		{"fibo-planner", "-h"},
		{"fibo-planner", "help"},
	} {
		t.Run(strings.Join(args[1:], " "), func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			cmd := newRootCommand()
			cmd.Writer = &out
			if err := cmd.Run(context.Background(), args); err != nil {
				t.Fatal(err)
			}
			text := out.String()
			for _, want := range []string{
				"fibo-planner",
				"real-time planning poker server",
				versioninfo.Version,
				"--help, -h",
				"--version, -v",
				"--addr string, -a string",
				"--ws-origins string",
				"--lobby-list-rooms",
				"--custom-html-dir string",
				"$" + listenAddrEnv,
				"$" + wsOriginsEnv,
				"$" + customHTMLDirEnv,
			} {
				if !strings.Contains(text, want) {
					t.Errorf("help missing %q\n%s", want, text)
				}
			}
		})
	}
}

func TestApplyServerFlags(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		env     map[string]string
		addr    string
		origins string
		list    bool
	}{
		{
			name:    "defaults",
			addr:    defaultListenAddr,
			origins: "",
			list:    false,
		},
		{
			name:    "addr from env",
			env:     map[string]string{listenAddrEnv: "  :9090  "},
			addr:    ":9090",
			origins: "",
		},
		{
			name:    "blank addr env keeps default",
			env:     map[string]string{listenAddrEnv: "   "},
			addr:    defaultListenAddr,
			origins: "",
		},
		{
			name:    "addr flag overrides env",
			args:    []string{"--addr", "127.0.0.1:3000"},
			env:     map[string]string{listenAddrEnv: ":9090"},
			addr:    "127.0.0.1:3000",
			origins: "",
		},
		{
			name:    "short addr flag",
			args:    []string{"-a", ":9090"},
			addr:    ":9090",
			origins: "",
		},
		{
			name:    "ws origins from env",
			env:     map[string]string{wsOriginsEnv: " https://app.example.com/,http://localhost:3000 "},
			addr:    defaultListenAddr,
			origins: "https://app.example.com/,http://localhost:3000",
		},
		{
			name:    "ws origins flag overrides env",
			args:    []string{"--ws-origins", "https://planner.example.com"},
			env:     map[string]string{wsOriginsEnv: "https://other.example"},
			addr:    defaultListenAddr,
			origins: "https://planner.example.com",
		},
		{
			name:    "lobby list from Y env",
			env:     map[string]string{lobbyListRoomsEnv: "Y"},
			addr:    defaultListenAddr,
			origins: "",
			list:    true,
		},
		{
			name:    "lobby list ignores non-Y env",
			env:     map[string]string{lobbyListRoomsEnv: "yes"},
			addr:    defaultListenAddr,
			origins: "",
			list:    false,
		},
		{
			name:    "lobby list flag overrides non-Y env",
			args:    []string{"--lobby-list-rooms"},
			env:     map[string]string{lobbyListRoomsEnv: "N"},
			addr:    defaultListenAddr,
			origins: "",
			list:    true,
		},
		{
			name:    "lobby list flag false overrides Y env",
			args:    []string{"--lobby-list-rooms=false"},
			env:     map[string]string{lobbyListRoomsEnv: "Y"},
			addr:    defaultListenAddr,
			origins: "",
			list:    false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(listenAddrEnv, "")
			t.Setenv(wsOriginsEnv, "")
			t.Setenv(lobbyListRoomsEnv, "")
			t.Setenv(customHTMLDirEnv, "")
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			t.Cleanup(func() {
				setAllowedWSOrigins("")
				if err := loadCustomContent(""); err != nil {
					t.Errorf("restore templates: %v", err)
				}
			})

			cmd := newRootCommand()
			cmd.Writer = io.Discard
			cmd.ErrWriter = io.Discard
			var addr string
			var list bool
			cmd.Action = func(_ context.Context, c *cli.Command) error {
				var err error
				addr, list, err = applyServerFlags(c)
				return err
			}
			args := append([]string{"fibo-planner"}, tc.args...)
			if err := cmd.Run(context.Background(), args); err != nil {
				t.Fatal(err)
			}
			if addr != tc.addr {
				t.Fatalf("addr=%q, want %q", addr, tc.addr)
			}
			if list != tc.list {
				t.Fatalf("listLobbyRooms=%v, want %v", list, tc.list)
			}
			if !slices.Equal(allowedWSOrigins, parseWSOrigins(tc.origins)) {
				t.Fatalf("origins=%q, want %q", allowedWSOrigins, parseWSOrigins(tc.origins))
			}
		})
	}
}

func TestCustomHTMLDirFlag(t *testing.T) {
	envDir := t.TempDir()
	flagDir := t.TempDir()
	writeSnippet(t, envDir, customHeadFile, `<!--ENV-HEAD-->`)
	writeSnippet(t, envDir, customLLMSFile, "from-env\n")
	writeSnippet(t, flagDir, customHeadFile, `<!--FLAG-HEAD-->`)
	writeSnippet(t, flagDir, customLLMSFile, "from-flag\n")

	t.Setenv(customHTMLDirEnv, envDir)
	t.Cleanup(func() {
		if err := loadCustomContent(""); err != nil {
			t.Errorf("restore templates: %v", err)
		}
	})

	run := func(args ...string) error {
		cmd := newRootCommand()
		cmd.Writer = io.Discard
		cmd.ErrWriter = io.Discard
		cmd.Action = func(_ context.Context, c *cli.Command) error {
			_, _, err := applyServerFlags(c)
			return err
		}
		return cmd.Run(context.Background(), append([]string{"fibo-planner"}, args...))
	}

	if err := run(); err != nil {
		t.Fatal(err)
	}
	page := executeNamed(t, tmpl, "index.html")
	if !strings.Contains(page, "<!--ENV-HEAD-->") {
		t.Fatal("env dir was not applied")
	}
	if string(llmsBody) != "from-env\n" {
		t.Fatalf("llms body %q", llmsBody)
	}

	if err := run("--custom-html-dir", flagDir); err != nil {
		t.Fatal(err)
	}
	page = executeNamed(t, tmpl, "index.html")
	if !strings.Contains(page, "<!--FLAG-HEAD-->") || strings.Contains(page, "<!--ENV-HEAD-->") {
		t.Fatal("flag did not override the env dir")
	}
	if string(llmsBody) != "from-flag\n" {
		t.Fatalf("llms body %q", llmsBody)
	}

	file := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run("--custom-html-dir", file); err == nil {
		t.Fatal("expected an error when the custom HTML path is not a directory")
	}
}

func TestRootCommandUnknownFlag(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	cmd.Writer = io.Discard
	cmd.ErrWriter = io.Discard
	if err := cmd.Run(context.Background(), []string{"fibo-planner", "--not-a-flag"}); err == nil {
		t.Fatal("expected an error for an unknown flag")
	}
}

func TestAccessLogURI(t *testing.T) {
	t.Parallel()

	cases := []struct {
		path, raw, want string
	}{
		{path: "/", want: "/"},
		{path: "/rooms", raw: "name=Ada", want: "/rooms?name=Ada"},
		{path: "/ws", want: "/ws"},
		{path: "/ws", raw: "name=Ada", want: "/ws"},
		{path: "/ws/123456", raw: "name=Ada&x=1", want: "/ws/123456?x=1"},
		{path: "/ws", raw: "%zz", want: "/ws"},
	}
	for _, tc := range cases {
		if got := accessLogURI(tc.path, tc.raw); got != tc.want {
			t.Errorf("accessLogURI(%q, %q)=%q, want %q", tc.path, tc.raw, got, tc.want)
		}
	}
}

func TestAccessLogEntry(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	httpReq := httptest.NewRequest(http.MethodGet, "http://planner.example/rooms?x=1", nil)
	httpReq.Host = "planner.example"
	httpReq.RemoteAddr = "10.0.0.1:1234"
	entry := accessLogFormatter{}.NewLogEntry(httpReq).(*accessLogEntry)
	if !strings.Contains(entry.msg, `"GET http://planner.example/rooms?x=1 HTTP/1.1" from 10.0.0.1:1234`) {
		t.Fatalf("http entry %q", entry.msg)
	}

	tlsReq := httptest.NewRequest(http.MethodGet, "https://planner.example/ws?name=Ada", nil)
	tlsReq.TLS = &tls.ConnectionState{}
	tlsReq.Host = "planner.example"
	tlsReq.RemoteAddr = "10.0.0.2:9"
	tlsEntry := accessLogFormatter{}.NewLogEntry(tlsReq).(*accessLogEntry)
	if !strings.Contains(tlsEntry.msg, `"GET https://planner.example/ws HTTP/1.1" from 10.0.0.2:9`) {
		t.Fatalf("https entry %q", tlsEntry.msg)
	}

	entry.Write(http.StatusOK, 12, nil, time.Millisecond, nil)
	tlsEntry.Panic("boom", []byte("stack"))
	text := buf.String()
	if !strings.Contains(text, "200 12B") || !strings.Contains(text, "panic: boom") || !strings.Contains(text, "stack") {
		t.Fatalf("log %q", text)
	}
}

func TestNewRouterServesHome(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(newRouter(newAppConfig(false)))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "create-room") {
		t.Fatalf("home status=%d body=%q", resp.StatusCode, body)
	}
}

func TestServeAction(t *testing.T) {
	t.Setenv(listenAddrEnv, "")
	t.Setenv(wsOriginsEnv, "")
	t.Setenv(lobbyListRoomsEnv, "")
	t.Setenv(customHTMLDirEnv, "")
	t.Cleanup(func() {
		setAllowedWSOrigins("")
		if err := loadCustomContent(""); err != nil {
			t.Errorf("restore templates: %v", err)
		}
	})

	run := func(args ...string) error {
		cmd := newRootCommand()
		cmd.Writer = io.Discard
		cmd.ErrWriter = io.Discard
		return cmd.Run(context.Background(), append([]string{"fibo-planner"}, args...))
	}
	if err := run("--addr", "127.0.0.1:99999"); err == nil {
		t.Fatal("expected listen error")
	}

	file := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run("--custom-html-dir", file); err == nil {
		t.Fatal("expected an error when the custom HTML path is not a directory")
	}
}

func TestVersionPrinterWriteError(t *testing.T) {
	orig := fatal
	t.Cleanup(func() { fatal = orig })
	var fataled any
	fatal = func(v ...any) {
		if len(v) > 0 {
			fataled = v[0]
		}
	}

	cmd := newRootCommand()
	cmd.Writer = errWriter{err: errors.New("write failed")}
	cmd.ErrWriter = io.Discard
	if err := cmd.Run(context.Background(), []string{"fibo-planner", "--version"}); err != nil {
		t.Fatal(err)
	}
	if fataled == nil {
		t.Fatal("expected fatal when version output fails")
	}
}

func TestProgramMain(t *testing.T) {
	origArgs := os.Args
	origStdout := os.Stdout
	origFatal := fatal
	t.Cleanup(func() {
		os.Args = origArgs
		os.Stdout = origStdout
		fatal = origFatal
	})

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(io.Discard, r)
		close(done)
	}()

	os.Args = []string{"fibo-planner", "--version"}
	main()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdout = origStdout
	<-done

	var fataled any
	fatal = func(v ...any) {
		if len(v) > 0 {
			fataled = v[0]
		}
	}
	os.Args = []string{"fibo-planner", "--not-a-flag"}
	main()
	if fataled == nil {
		t.Fatal("expected fatal for an unknown flag")
	}
}

type errWriter struct{ err error }

func (w errWriter) Write([]byte) (int, error) { return 0, w.err }
