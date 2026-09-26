package versioninfo

import "testing"

func TestFormat(t *testing.T) {
	origV, origD, origC := Version, Date, Commit
	t.Cleanup(func() {
		Version, Date, Commit = origV, origD, origC
	})

	Version = "1.2.3"
	Commit = "abc123"
	Date = "2026-09-26T10:00:00Z"

	got := Format()
	want := "Version: 1.2.3\nCommit: abc123\nBuildDate: 2026-09-26T10:00:00Z\n"
	if got != want {
		t.Fatalf("Format()=%q, want %q", got, want)
	}
}
