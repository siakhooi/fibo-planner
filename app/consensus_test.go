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
