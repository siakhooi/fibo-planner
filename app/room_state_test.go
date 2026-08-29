package main

import (
	"strings"
	"testing"
)

func TestRoomStateHTML(t *testing.T) {
	t.Parallel()

	html := roomStateHTML(2, []participant{
		{name: "Bob", points: "8"},
		{name: "Ada", points: "3"},
	})
	if !strings.Contains(html, `id="user-list"`) {
		t.Fatalf("missing user-list table: %s", html)
	}
	if !strings.Contains(html, "Points") {
		t.Fatalf("missing Points column: %s", html)
	}
	if !strings.Contains(html, `<strong id="session-count" hx-swap-oob="true">2</strong>`) {
		t.Fatalf("missing session count: %s", html)
	}
	ada := strings.Index(html, "Ada")
	bob := strings.Index(html, "Bob")
	if ada < 0 || bob < 0 || ada > bob {
		t.Fatalf("rows should be sorted by name: %s", html)
	}
	if !strings.Contains(html, "<td>3</td>") || !strings.Contains(html, "<td>8</td>") {
		t.Fatalf("missing points cells: %s", html)
	}

	escaped := roomStateHTML(1, []participant{{name: "<script>alert(1)</script>", points: "1"}})
	if strings.Contains(escaped, "<script>") {
		t.Fatalf("name was not escaped: %s", escaped)
	}
	if !strings.Contains(escaped, "&lt;script&gt;") {
		t.Fatalf("expected escaped name: %s", escaped)
	}
}
func TestRoomStateHTMLMasksPointsUntilEveryoneVoted(t *testing.T) {
	t.Parallel()

	html := roomStateHTML(2, []participant{
		{name: "Ada", points: "8"},
		{name: "Bob", points: ""},
	})
	if !strings.Contains(html, "<td>Ada</td><td>???</td>") {
		t.Fatalf("Ada's vote should be masked: %s", html)
	}
	if strings.Contains(html, "<td>Ada</td><td>8</td>") {
		t.Fatalf("Ada's actual points leaked: %s", html)
	}
	if !strings.Contains(html, "<td>Bob</td><td></td>") {
		t.Fatalf("Bob's empty vote should stay empty: %s", html)
	}
}

func TestRoomStateHTMLRevealsPointsWhenEveryoneVoted(t *testing.T) {
	t.Parallel()

	html := roomStateHTML(2, []participant{
		{name: "Ada", points: "8"},
		{name: "Bob", points: "5"},
	})
	if strings.Contains(html, "???") {
		t.Fatalf("points should be visible once everyone voted: %s", html)
	}
	if !strings.Contains(html, "<td>Ada</td><td>8</td>") || !strings.Contains(html, "<td>Bob</td><td>5</td>") {
		t.Fatalf("missing revealed points: %s", html)
	}
}
