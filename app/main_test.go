package main

import (
	"bytes"
	"context"
	"io"
	"slices"
	"strings"
	"testing"

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
				"$" + listenAddrEnv,
				"$" + wsOriginsEnv,
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
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			t.Cleanup(func() { setAllowedWSOrigins("") })

			cmd := newRootCommand()
			cmd.Writer = io.Discard
			cmd.ErrWriter = io.Discard
			var addr string
			var list bool
			cmd.Action = func(_ context.Context, c *cli.Command) error {
				addr, list = applyServerFlags(c)
				return nil
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

func TestRootCommandUnknownFlag(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	cmd.Writer = io.Discard
	cmd.ErrWriter = io.Discard
	if err := cmd.Run(context.Background(), []string{"fibo-planner", "--not-a-flag"}); err == nil {
		t.Fatal("expected an error for an unknown flag")
	}
}
