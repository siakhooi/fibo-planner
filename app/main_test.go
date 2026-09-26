package main

import "testing"

func TestHasVersionFlag(t *testing.T) {
	t.Parallel()

	cases := []struct {
		args []string
		want bool
	}{
		{args: nil, want: false},
		{args: []string{}, want: false},
		{args: []string{"--version"}, want: true},
		{args: []string{"-version"}, want: false},
		{args: []string{"--help"}, want: false},
		{args: []string{"--version", "extra"}, want: true},
	}
	for _, tc := range cases {
		if got := hasVersionFlag(tc.args); got != tc.want {
			t.Errorf("hasVersionFlag(%q)=%v, want %v", tc.args, got, tc.want)
		}
	}
}
