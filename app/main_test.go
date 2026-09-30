package main

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/siakhooi/fibo-planner/app/versioninfo"
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
			} {
				if !strings.Contains(text, want) {
					t.Errorf("help missing %q\n%s", want, text)
				}
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
