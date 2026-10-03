package main

import "testing"

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
	assertTallies(t, got, want)

	tied := tallyVotes([]participant{
		{points: "13"},
		{points: "3"},
		{points: "13"},
		{points: "3"},
		{points: "8"},
	})
	assertTallies(t, tied, []voteTally{
		{points: "3", count: 2},
		{points: "13", count: 2},
		{points: "8", count: 1},
	})
}

func assertTallies(t *testing.T, got, want []voteTally) {
	t.Helper()
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
		{1, -1, 0},
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

func TestMeetsConsensus(t *testing.T) {
	t.Parallel()

	cases := []struct {
		percent, threshold int
		want               bool
	}{
		{50, 50, false},
		{51, 50, true},
		{51, 40, true},
		{100, 50, true},
		{74, 75, false},
		{75, 75, true},
		{100, 75, true},
		{99, 100, false},
		{100, 100, true},
		{99, 110, false},
		{100, 110, true},
	}
	for _, c := range cases {
		if got := meetsConsensus(c.percent, c.threshold); got != c.want {
			t.Fatalf("meetsConsensus(%d, %d) = %v, want %v", c.percent, c.threshold, got, c.want)
		}
	}
}

func TestVoteSpread(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		rows []participant
		want int
	}{
		{name: "3 and 5", rows: []participant{{points: "3"}, {points: "5"}}, want: 1},
		{name: "3 and 8", rows: []participant{{points: "3"}, {points: "8"}}, want: 2},
		{name: "20 and 1", rows: []participant{{points: "20"}, {points: "1"}}, want: 6},
		{name: "unanimous", rows: []participant{{points: "8"}, {points: "8"}}, want: 0},
		{name: "ignores observers", rows: []participant{{points: "3"}, {points: "20", observer: true}}, want: 0},
		{name: "ignores empty", rows: []participant{{points: "5"}, {points: ""}}, want: 0},
		{name: "ignores unknown", rows: []participant{{points: "nope"}, {points: "5"}, {points: "8"}}, want: 1},
		{name: "no votes", rows: nil, want: 0},
		{name: "only observers", rows: []participant{{points: "20", observer: true}}, want: 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := voteSpread(c.rows); got != c.want {
				t.Fatalf("voteSpread = %d, want %d", got, c.want)
			}
		})
	}
}

func TestPointsRank(t *testing.T) {
	t.Parallel()

	for i, p := range voteScale {
		rank, ok := pointsRank(p)
		if !ok || rank != i {
			t.Fatalf("pointsRank(%q) = %d, %v; want %d, true", p, rank, ok, i)
		}
	}
	if rank, ok := pointsRank(""); ok {
		t.Fatalf("pointsRank(\"\") = %d, true", rank)
	}
	if rank, ok := pointsRank("nope"); ok {
		t.Fatalf("pointsRank(\"nope\") = %d, true", rank)
	}
}

func TestNormalizeConsensusPercent(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in, want int
	}{
		{minConsensusPercent - 1, defaultConsensusPercent},
		{maxConsensusPercent + 1, defaultConsensusPercent},
		{minConsensusPercent, minConsensusPercent},
		{75, 75},
		{maxConsensusPercent, maxConsensusPercent},
	}
	for _, c := range cases {
		if got := normalizeConsensusPercent(c.in); got != c.want {
			t.Fatalf("normalizeConsensusPercent(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestNormalizeMaxSpread(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in, want int
	}{
		{minMaxSpread - 1, defaultMaxSpread},
		{maxMaxSpread + 1, defaultMaxSpread},
		{minMaxSpread, minMaxSpread},
		{3, 3},
		{maxMaxSpread, maxMaxSpread},
	}
	for _, c := range cases {
		if got := normalizeMaxSpread(c.in); got != c.want {
			t.Fatalf("normalizeMaxSpread(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestAllVotersHaveVoted(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		rows []participant
		want bool
	}{
		{name: "empty", rows: nil, want: false},
		{name: "only observers", rows: []participant{{observer: true, points: "5"}}, want: false},
		{name: "blank vote", rows: []participant{{points: ""}}, want: false},
		{name: "observer then blank", rows: []participant{{observer: true, points: "8"}, {points: ""}}, want: false},
		{name: "one voter", rows: []participant{{points: "5"}}, want: true},
		{name: "observer ignored", rows: []participant{{observer: true}, {points: "8"}}, want: true},
		{name: "one blank among voters", rows: []participant{{points: "5"}, {points: ""}}, want: false},
		{name: "everyone voted", rows: []participant{{points: "5"}, {points: "8"}}, want: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := allVotersHaveVoted(c.rows); got != c.want {
				t.Fatalf("allVotersHaveVoted = %v, want %v", got, c.want)
			}
		})
	}
}

func TestMeetsSpread(t *testing.T) {
	t.Parallel()

	cases := []struct {
		spread, maxSpread int
		want              bool
	}{
		{0, 0, true},
		{1, 0, false},
		{2, 2, true},
		{3, 2, false},
	}
	for _, c := range cases {
		if got := meetsSpread(c.spread, c.maxSpread); got != c.want {
			t.Fatalf("meetsSpread(%d, %d) = %v, want %v", c.spread, c.maxSpread, got, c.want)
		}
	}
}

func TestAgreedPoints(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name                                string
		tallies                             []voteTally
		total, consensus, spread, maxSpread int
		want                                string
	}{
		{
			name:      "spread too wide",
			tallies:   []voteTally{{points: "8", count: 2}},
			total:     2,
			consensus: 100,
			spread:    2,
			maxSpread: 1,
			want:      "N/A",
		},
		{
			name:      "nobody meets the bar",
			tallies:   []voteTally{{points: "5", count: 1}, {points: "8", count: 1}},
			total:     3,
			consensus: 100,
			spread:    1,
			maxSpread: 1,
			want:      "N/A",
		},
		{
			name:      "one value",
			tallies:   []voteTally{{points: "8", count: 2}, {points: "5", count: 1}},
			total:     2,
			consensus: 100,
			spread:    1,
			maxSpread: 1,
			want:      "8",
		},
		{
			name:      "several values",
			tallies:   []voteTally{{points: "8", count: 2}, {points: "13", count: 2}},
			total:     2,
			consensus: 100,
			spread:    1,
			maxSpread: 2,
			want:      "8, 13",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got := agreedPoints(c.tallies, c.total, c.consensus, c.spread, c.maxSpread)
			if got != c.want {
				t.Fatalf("agreedPoints = %q, want %q", got, c.want)
			}
		})
	}
}
