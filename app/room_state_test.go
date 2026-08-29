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
	}, false)
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

	escaped := roomStateHTML(1, []participant{{name: "<script>alert(1)</script>", points: "1"}}, false)
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
	}, false)
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
	}, false)
	if strings.Contains(html, "???") {
		t.Fatalf("points should be visible once everyone voted: %s", html)
	}
	if !strings.Contains(html, "<td>Ada</td><td>8</td>") || !strings.Contains(html, "<td>Bob</td><td>5</td>") {
		t.Fatalf("missing revealed points: %s", html)
	}
}

func TestRoomStateHTMLFlashesTheVotedCell(t *testing.T) {
	t.Parallel()

	html := roomStateHTML(2, []participant{
		{name: "Ada", points: "8", flash: true},
		{name: "Bob", points: ""},
	}, false)
	if !strings.Contains(html, `<td>Ada</td><td class="vote-flash">???</td>`) {
		t.Fatalf("Ada's masked vote should flash: %s", html)
	}
	if strings.Contains(html, `Bob</td><td class="vote-flash"`) {
		t.Fatalf("Bob should not flash: %s", html)
	}
}

func TestRoomStateHTMLAlwaysShowVotesSkipsMasking(t *testing.T) {
	t.Parallel()

	html := roomStateHTML(2, []participant{
		{name: "Ada", points: "8"},
		{name: "Bob", points: ""},
	}, true)
	if !strings.Contains(html, "<td>Ada</td><td>8</td>") {
		t.Fatalf("Ada's vote should be visible: %s", html)
	}
	if strings.Contains(html, "???") {
		t.Fatalf("votes should not be masked: %s", html)
	}
	if !strings.Contains(html, `id="always-show-votes" hx-swap-oob="true" aria-pressed="true"`) {
		t.Fatalf("always-show button should be pressed: %s", html)
	}
}

func TestRoomStateHTMLObserversListedLastAndSkipMasking(t *testing.T) {
	t.Parallel()

	html := roomStateHTML(3, []participant{
		{name: "Zed", observer: true, points: "8"},
		{name: "Ada", points: "5"},
		{name: "Bob", observer: true},
	}, false)
	if strings.Contains(html, "???") {
		t.Fatalf("observer should not keep votes masked: %s", html)
	}
	if !strings.Contains(html, "<td>Ada</td><td>5</td>") {
		t.Fatalf("Ada's vote should be revealed: %s", html)
	}
	ada := strings.Index(html, "<td>Ada</td>")
	zed := strings.Index(html, "<td>Zed</td>")
	bob := strings.Index(html, "<td>Bob</td>")
	if ada < 0 || zed < 0 || bob < 0 || ada > bob || bob > zed {
		t.Fatalf("voters then observers by name, want Ada then Bob, Zed: %s", html)
	}
	if !strings.Contains(html, "<td>Zed</td><td>observer</td>") || !strings.Contains(html, "<td>Bob</td><td>observer</td>") {
		t.Fatalf("observer cells should say observer: %s", html)
	}
	if strings.Contains(html, "<td>Zed</td><td>8</td>") {
		t.Fatalf("observer previous vote should not show: %s", html)
	}
}
