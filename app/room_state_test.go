package main

import (
	"strings"
	"testing"
)

func roomHTML(n int, rows []participant, alwaysShow bool) string {
	return roomStateHTML(n, rows, alwaysShow, "", defaultConsensusPercent)
}

func TestRoomStateHTML(t *testing.T) {
	t.Parallel()

	html := roomHTML(2, []participant{
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
	if !strings.Contains(html, `step="1" value="100"`) {
		t.Fatalf("consensus slider should default to 100: %s", html)
	}
	ada := strings.Index(html, "Ada")
	bob := strings.Index(html, "Bob")
	if ada < 0 || bob < 0 || ada > bob {
		t.Fatalf("rows should be sorted by name: %s", html)
	}
	if !strings.Contains(html, "<td>3</td>") || !strings.Contains(html, "<td>8</td>") {
		t.Fatalf("missing points cells: %s", html)
	}

	escaped := roomHTML(1, []participant{{name: "<script>alert(1)</script>", points: "1"}}, false)
	if strings.Contains(escaped, "<script>") {
		t.Fatalf("name was not escaped: %s", escaped)
	}
	if !strings.Contains(escaped, "&lt;script&gt;") {
		t.Fatalf("expected escaped name: %s", escaped)
	}
}
func TestRoomStateHTMLMasksPointsUntilEveryoneVoted(t *testing.T) {
	t.Parallel()

	html := roomHTML(2, []participant{
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

	html := roomHTML(2, []participant{
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

	html := roomHTML(2, []participant{
		{name: "Ada", points: "8", flash: true},
		{name: "Bob", points: ""},
	}, false)
	if !strings.Contains(html, `<td class="vote-flash">Ada</td><td class="vote-flash">???</td>`) {
		t.Fatalf("Ada's masked vote should flash: %s", html)
	}
	if strings.Contains(html, `Bob</td><td class="vote-flash"`) || strings.Contains(html, `class="vote-flash">Bob`) {
		t.Fatalf("Bob should not flash: %s", html)
	}
}

func TestRoomStateHTMLFlashesObserverRoleChange(t *testing.T) {
	t.Parallel()

	html := roomHTML(2, []participant{
		{name: "Ada", points: "8"},
		{name: "Bob", observer: true, flash: true},
	}, false)
	if !strings.Contains(html, `<td class="vote-flash">Bob</td><td class="vote-flash">observer</td>`) {
		t.Fatalf("Bob's observer switch should flash: %s", html)
	}
	if strings.Contains(html, `class="vote-flash">Ada`) {
		t.Fatalf("Ada should not flash: %s", html)
	}
}

func TestRoomStateHTMLAlwaysShowVotesSkipsMasking(t *testing.T) {
	t.Parallel()

	html := roomHTML(2, []participant{
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

	html := roomHTML(3, []participant{
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

const voteResultsHiddenMarkup = `id="vote-results" class="user-table results-table" hx-swap-oob="true" hidden`
const voteResultsVisibleMarkup = `id="vote-results" class="user-table results-table" hx-swap-oob="true">`

func TestRoomStateHTMLHidesResultsUntilEveryoneVoted(t *testing.T) {
	t.Parallel()

	html := roomHTML(2, []participant{
		{name: "Ada", points: "8"},
		{name: "Bob", points: ""},
	}, false)
	if !strings.Contains(html, voteResultsHiddenMarkup) {
		t.Fatalf("results table should be hidden: %s", html)
	}
	if strings.Contains(html, `<td>8</td><td>1</td>`) {
		t.Fatalf("results rows should not be sent before everyone voted: %s", html)
	}
}

func TestRoomStateHTMLHidesResultsWhenAlwaysShowButNotEveryoneVoted(t *testing.T) {
	t.Parallel()

	html := roomHTML(2, []participant{
		{name: "Ada", points: "8"},
		{name: "Bob", points: ""},
	}, true)
	if !strings.Contains(html, voteResultsHiddenMarkup) {
		t.Fatalf("results should stay hidden until everyone voted: %s", html)
	}
}

func TestRoomStateHTMLHidesResultsWhenOnlyObservers(t *testing.T) {
	t.Parallel()

	html := roomHTML(1, []participant{
		{name: "Ada", observer: true},
	}, false)
	if !strings.Contains(html, voteResultsHiddenMarkup) {
		t.Fatalf("results should be hidden with no voters: %s", html)
	}
}

func TestRoomStateHTMLShowsResultsOrderedByCount(t *testing.T) {
	t.Parallel()

	html := roomHTML(4, []participant{
		{name: "Ada", points: "8"},
		{name: "Bob", points: "5"},
		{name: "Cyd", points: "8"},
		{name: "Dee", points: "3"},
	}, false)
	if !strings.Contains(html, voteResultsVisibleMarkup) {
		t.Fatalf("results table should be visible: %s", html)
	}
	if strings.Contains(html, voteResultsHiddenMarkup) {
		t.Fatalf("results table should not be hidden: %s", html)
	}
	eight := strings.Index(html, `<tr class="vote-leader"><td>8</td><td>2</td><td>50%</td></tr>`)
	three := strings.Index(html, `<tr><td>3</td><td>1</td><td>25%</td></tr>`)
	five := strings.Index(html, `<tr><td>5</td><td>1</td><td>25%</td></tr>`)
	if eight < 0 || five < 0 || three < 0 {
		t.Fatalf("missing result rows: %s", html)
	}
	if eight > three || three > five {
		t.Fatalf("want count desc then points asc, got html: %s", html)
	}
	if strings.Contains(html, `class="vote-leader"><td>5</td>`) || strings.Contains(html, `class="vote-leader"><td>3</td>`) {
		t.Fatalf("only the highest count should be highlighted: %s", html)
	}
}

func TestRoomStateHTMLHighlightsTiedHighestCounts(t *testing.T) {
	t.Parallel()

	html := roomHTML(4, []participant{
		{name: "Ada", points: "8"},
		{name: "Bob", points: "5"},
		{name: "Cyd", points: "8"},
		{name: "Dee", points: "5"},
	}, false)
	five := strings.Index(html, `<tr class="vote-leader"><td>5</td><td>2</td><td>50%</td></tr>`)
	eight := strings.Index(html, `<tr class="vote-leader"><td>8</td><td>2</td><td>50%</td></tr>`)
	if five < 0 || eight < 0 {
		t.Fatalf("both tied leaders should be highlighted: %s", html)
	}
	if five > eight {
		t.Fatalf("tied counts should list lower points first: %s", html)
	}
}

func TestRoomStateHTMLResultsIgnoreObservers(t *testing.T) {
	t.Parallel()

	html := roomHTML(3, []participant{
		{name: "Ada", points: "5"},
		{name: "Bob", observer: true},
		{name: "Zed", observer: true, points: "8"},
	}, false)
	if !strings.Contains(html, voteResultsVisibleMarkup) {
		t.Fatalf("Ada is the only voter and has voted, results should show: %s", html)
	}
	if !strings.Contains(html, `<tr class="vote-leader"><td>5</td><td>1</td><td>100%</td></tr>`) {
		t.Fatalf("results should count only Ada: %s", html)
	}
	if strings.Contains(html, `<td>8</td><td>1</td>`) {
		t.Fatalf("observer leftover points must not be tallied: %s", html)
	}
}

func TestTallyVotes(t *testing.T) {
	t.Parallel()

	got := tallyVotes([]participant{
		{name: "A", points: "13"},
		{name: "B", points: "8"},
		{name: "C", points: "8"},
		{name: "D", observer: true, points: "20"},
		{name: "E", points: ""},
		{name: "F", points: "13"},
		{name: "G", points: "8"},
	})
	want := []voteTally{
		{points: "8", count: 3},
		{points: "13", count: 2},
	}
	if len(got) != len(want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("index %d: got %#v, want %#v", i, got, want)
		}
	}
}

func TestPercentOf(t *testing.T) {
	t.Parallel()

	cases := []struct {
		count, total, want int
	}{
		{0, 0, 0},
		{1, 1, 100},
		{1, 2, 50},
		{1, 3, 33},
		{2, 3, 67},
		{1, 4, 25},
		{2, 4, 50},
	}
	for _, c := range cases {
		if got := percentOf(c.count, c.total); got != c.want {
			t.Fatalf("percentOf(%d, %d) = %d, want %d", c.count, c.total, got, c.want)
		}
	}
}

func TestRoomStateHTMLHidesEmptyTopic(t *testing.T) {
	t.Parallel()

	html := roomHTML(1, []participant{{name: "Ada"}}, false)
	if !strings.Contains(html, `<h2 id="topic-title" class="topic-title" hx-swap-oob="true" hidden></h2>`) {
		t.Fatalf("empty topic should be hidden: %s", html)
	}
}

func TestRoomStateHTMLShowsEscapedTopic(t *testing.T) {
	t.Parallel()

	html := roomStateHTML(1, []participant{{name: "Ada", points: "1"}}, false, "<script>x</script>", defaultConsensusPercent)
	if !strings.Contains(html, `<h2 id="topic-title" class="topic-title" hx-swap-oob="true">&lt;script&gt;x&lt;/script&gt;</h2>`) {
		t.Fatalf("topic should be visible and escaped: %s", html)
	}
	if strings.Contains(html, `id="topic-title" class="topic-title" hx-swap-oob="true" hidden`) {
		t.Fatalf("topic heading should not be hidden: %s", html)
	}
}

func TestMeetsConsensus(t *testing.T) {
	t.Parallel()

	cases := []struct {
		percent, threshold int
		want               bool
	}{
		{50, 50, false},
		{51, 50, true},
		{100, 50, true},
		{74, 75, false},
		{75, 75, true},
		{100, 75, true},
		{99, 100, false},
		{100, 100, true},
	}
	for _, c := range cases {
		if got := meetsConsensus(c.percent, c.threshold); got != c.want {
			t.Fatalf("meetsConsensus(%d, %d) = %v, want %v", c.percent, c.threshold, got, c.want)
		}
	}
}

func TestRoomStateHTMLHidesAgreedPointsUntilEveryoneVoted(t *testing.T) {
	t.Parallel()

	html := roomHTML(2, []participant{
		{name: "Ada", points: "8"},
		{name: "Bob", points: ""},
	}, false)
	if !strings.Contains(html, `<p id="agreed-points" hx-swap-oob="true" hidden></p>`) {
		t.Fatalf("agreed points should stay hidden: %s", html)
	}
	if strings.Contains(html, "Agreed Points:") {
		t.Fatalf("agreed points leaked before everyone voted: %s", html)
	}
}

func TestRoomStateHTMLShowsAgreedPointsWhenShareMeetsThreshold(t *testing.T) {
	t.Parallel()

	rows := []participant{
		{name: "Ada", points: "8"},
		{name: "Bob", points: "8"},
		{name: "Cyd", points: "5"},
	}
	html := roomStateHTML(3, rows, false, "", 50)
	if !strings.Contains(html, `<p id="agreed-points" class="agreed-yes" hx-swap-oob="true">Agreed Points: <strong>8</strong></p>`) {
		t.Fatalf("67%% is > 50, want agreed 8: %s", html)
	}

	html = roomStateHTML(3, rows, false, "", 67)
	if !strings.Contains(html, `class="agreed-yes"`) || !strings.Contains(html, `Agreed Points: <strong>8</strong>`) {
		t.Fatalf("67%% is >= 67, want agreed 8: %s", html)
	}

	html = roomStateHTML(3, rows, false, "", 68)
	if !strings.Contains(html, `<p id="agreed-points" class="agreed-no" hx-swap-oob="true">Agreed Points: <strong>N/A</strong></p>`) {
		t.Fatalf("67%% is not >= 68, want N/A: %s", html)
	}
}

func TestRoomStateHTMLFiftyMeansStrictMajority(t *testing.T) {
	t.Parallel()

	html := roomStateHTML(2, []participant{
		{name: "Ada", points: "8"},
		{name: "Bob", points: "5"},
	}, false, "", 50)
	if !strings.Contains(html, `<p id="agreed-points" class="agreed-no" hx-swap-oob="true">Agreed Points: <strong>N/A</strong></p>`) {
		t.Fatalf("50%% split should show N/A at threshold 50: %s", html)
	}
}

func TestRoomStateHTMLHundredMeansUnanimous(t *testing.T) {
	t.Parallel()

	split := roomStateHTML(2, []participant{
		{name: "Ada", points: "8"},
		{name: "Bob", points: "5"},
	}, false, "", 100)
	if !strings.Contains(split, `<p id="agreed-points" class="agreed-no" hx-swap-oob="true">Agreed Points: <strong>N/A</strong></p>`) {
		t.Fatalf("split vote is not 100%%, want N/A: %s", split)
	}

	unanimous := roomStateHTML(2, []participant{
		{name: "Ada", points: "8"},
		{name: "Bob", points: "8"},
	}, false, "", 100)
	if !strings.Contains(unanimous, `<p id="agreed-points" class="agreed-yes" hx-swap-oob="true">Agreed Points: <strong>8</strong></p>`) {
		t.Fatalf("unanimous 8 should agree at 100: %s", unanimous)
	}
}

func TestRoomStateHTMLSyncsConsensusSlider(t *testing.T) {
	t.Parallel()

	html := roomStateHTML(1, []participant{{name: "Ada", points: "1"}}, false, "", 80)
	if !strings.Contains(html, `id="consensus-percent"`) {
		t.Fatalf("missing consensus slider: %s", html)
	}
	if !strings.Contains(html, `value="80"`) {
		t.Fatalf("slider should be 80: %s", html)
	}
	if !strings.Contains(html, `>80</output>`) {
		t.Fatalf("percentage readout should be 80: %s", html)
	}
}
