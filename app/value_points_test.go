package main

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestParseVotePoints(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload string
		want    string
		ok      bool
	}{
		{name: "string five", payload: `{"points":"5"}`, want: "5", ok: true},
		{name: "clear vote", payload: `{"points":"","HEADERS":{}}`, want: "", ok: true},
		{name: "numeric thirteen", payload: `{"points":13}`, want: "13", ok: true},
		{name: "twenty", payload: `{"points":"20"}`, want: "20", ok: true},
		{name: "not json", payload: `points=5`, ok: false},
		{name: "unknown value", payload: `{"points":"99"}`, ok: false},
		{name: "missing points", payload: `{"HEADERS":{}}`, ok: false},
		{name: "script", payload: `{"points":"<script>"}`, ok: false},
		{name: "admin is not a vote", payload: `{"admin":"set-topic"}`, ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := parseVotePoints([]byte(tt.payload))
			if ok != tt.ok {
				t.Fatalf("ok=%v want %v", ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Fatalf("points=%q want %q", got, tt.want)
			}
		})
	}
}

func TestParseAdminAction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload string
		want    string
		ok      bool
	}{
		{name: "clear votes", payload: `{"admin":"clear-votes","HEADERS":{}}`, want: adminClearVotes, ok: true},
		{name: "set topic", payload: `{"admin":"set-topic","HEADERS":{}}`, want: adminSetTopic, ok: true},
		{name: "always show", payload: `{"admin":"always-show-votes"}`, want: adminAlwaysShowVotes, ok: true},
		{name: "observer", payload: `{"admin":"observer-mode"}`, want: adminObserverMode, ok: true},
		{name: "consensus", payload: `{"admin":"consensus-agreement","percentage":"75"}`, want: adminConsensusAgreement, ok: true},
		{name: "load next topic", payload: `{"admin":"load-next-topic"}`, want: adminLoadNextTopic, ok: true},
		{name: "set preloaded topics", payload: `{"admin":"set-preloaded-topics","preloaded-topics":"A"}`, want: adminSetPreloadedTopics, ok: true},
		{name: "unknown", payload: `{"admin":"explode"}`, ok: false},
		{name: "vote", payload: `{"points":"8"}`, ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := parseAdminAction([]byte(tt.payload))
			if ok != tt.ok {
				t.Fatalf("ok=%v want %v", ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Fatalf("action=%q want %q", got, tt.want)
			}
		})
	}
}

func TestParseJoinName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload string
		want    string
		ok      bool
	}{
		{name: "plain", payload: `{"name":"Ada"}`, want: "Ada", ok: true},
		{name: "trimmed", payload: `{"name":"  Bob  "}`, want: "Bob", ok: true},
		{name: "empty", payload: `{"name":"   "}`, want: "", ok: true},
		{name: "htmx headers", payload: `{"name":"Cyd","HEADERS":{}}`, want: "Cyd", ok: true},
		{name: "vote", payload: `{"points":"8"}`, ok: false},
		{name: "admin", payload: `{"admin":"clear-votes"}`, ok: false},
		{name: "numeric", payload: `{"name":1}`, ok: false},
		{name: "not json", payload: `name=Ada`, ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := parseJoinName([]byte(tt.payload))
			if ok != tt.ok {
				t.Fatalf("ok=%v want %v", ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Fatalf("name=%q want %q", got, tt.want)
			}
		})
	}
}

func TestParseTopicTitle(t *testing.T) {
	t.Parallel()

	if got := parseTopicTitle([]byte(`{"admin":"set-topic","topic-title":" Login "}`)); got != "Login" {
		t.Fatalf("got title=%q", got)
	}
	if got := parseTopicTitle([]byte(`{"admin":"set-topic"}`)); got != "" {
		t.Fatalf("missing title should be empty, got %q", got)
	}
}

func TestParsePreloadedTopics(t *testing.T) {
	t.Parallel()

	got := parsePreloadedTopics([]byte("{\"admin\":\"set-preloaded-topics\",\"preloaded-topics\":\"  Alpha  \\n\\nBeta\\n  \\nGamma  \"}"))
	want := []string{"Alpha", "Beta", "Gamma"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}

	crlf := parsePreloadedTopics([]byte("{\"preloaded-topics\":\"One\\r\\n\\r\\nTwo\"}"))
	if len(crlf) != 2 || crlf[0] != "One" || crlf[1] != "Two" {
		t.Fatalf("CRLF should split and drop blanks: %v", crlf)
	}

	long := strings.Repeat("a", maxTopicTitleLen+5)
	trimmed := normalizePreloadedTopics(long)
	if len(trimmed) != 1 || trimmed[0] != strings.Repeat("a", maxTopicTitleLen) {
		t.Fatalf("title should truncate to %d, got %#v", maxTopicTitleLen, trimmed)
	}

	multibyte := strings.Repeat("é", maxTopicTitleLen+1)
	trimmed = normalizePreloadedTopics(multibyte)
	if len(trimmed) != 1 || trimmed[0] != strings.Repeat("é", maxTopicTitleLen) {
		t.Fatalf("multibyte title should truncate to %d runes, got %#v", maxTopicTitleLen, trimmed)
	}
	if !utf8.ValidString(trimmed[0]) {
		t.Fatal("truncated multibyte title must be valid UTF-8")
	}

	if got := parsePreloadedTopics([]byte(`{"admin":"set-preloaded-topics"}`)); len(got) != 0 {
		t.Fatalf("missing field should be empty, got %v", got)
	}

	var b strings.Builder
	for i := 0; i < maxPreloadedTopicCount+3; i++ {
		b.WriteString("t\n")
	}
	capped := normalizePreloadedTopics(b.String())
	if len(capped) != maxPreloadedTopicCount {
		t.Fatalf("got %d topics, want cap %d", len(capped), maxPreloadedTopicCount)
	}
}

func TestParseConsensusPercent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload string
		want    int
		ok      bool
	}{
		{name: "string seventy five", payload: `{"admin":"consensus-agreement","percentage":"75"}`, want: 75, ok: true},
		{name: "numeric fifty", payload: `{"admin":"consensus-agreement","percentage":50}`, want: 50, ok: true},
		{name: "hundred", payload: `{"percentage":"100"}`, want: 100, ok: true},
		{name: "below min", payload: `{"percentage":"49"}`, ok: false},
		{name: "above max", payload: `{"percentage":"101"}`, ok: false},
		{name: "not an integer", payload: `{"percentage":"75.5"}`, ok: false},
		{name: "missing", payload: `{"admin":"consensus-agreement"}`, ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := parseConsensusPercent([]byte(tt.payload))
			if ok != tt.ok {
				t.Fatalf("ok=%v want %v", ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Fatalf("percent=%d want %d", got, tt.want)
			}
		})
	}
}

func TestParseMaxSpread(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		payload string
		want    int
		ok      bool
	}{
		{name: "string two", payload: `{"admin":"consensus-agreement","max-spread":"2"}`, want: 2, ok: true},
		{name: "numeric zero", payload: `{"admin":"consensus-agreement","max-spread":0}`, want: 0, ok: true},
		{name: "six", payload: `{"max-spread":"6"}`, want: 6, ok: true},
		{name: "below min", payload: `{"max-spread":"-1"}`, ok: false},
		{name: "above max", payload: `{"max-spread":"7"}`, ok: false},
		{name: "not an integer", payload: `{"max-spread":"1.5"}`, ok: false},
		{name: "missing", payload: `{"admin":"consensus-agreement","percentage":"75"}`, ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := parseMaxSpread([]byte(tt.payload))
			if ok != tt.ok {
				t.Fatalf("ok=%v want %v", ok, tt.ok)
			}
			if ok && got != tt.want {
				t.Fatalf("spread=%d want %d", got, tt.want)
			}
		})
	}
}

func TestTruncateRunes(t *testing.T) {
	t.Parallel()

	if got := truncateRunes("hello", 10); got != "hello" {
		t.Fatalf("short string: got %q", got)
	}
	if got := truncateRunes(strings.Repeat("a", maxDisplayNameLen+5), maxDisplayNameLen); got != strings.Repeat("a", maxDisplayNameLen) {
		t.Fatalf("ascii should cut to %d letters", maxDisplayNameLen)
	}

	long := strings.Repeat("é", maxDisplayNameLen+1)
	got := truncateRunes(long, maxDisplayNameLen)
	if got != strings.Repeat("é", maxDisplayNameLen) {
		t.Fatalf("got %d runes, want %d", utf8.RuneCountInString(got), maxDisplayNameLen)
	}
	if !utf8.ValidString(got) {
		t.Fatal("truncated string must be valid UTF-8")
	}

	mixed := strings.Repeat("a", maxDisplayNameLen-1) + "你"
	if got := truncateRunes(mixed, maxDisplayNameLen); got != mixed {
		t.Fatalf("should keep 119 ascii + one CJK, got %q", got)
	}
	if utf8.ValidString(mixed[:maxDisplayNameLen]) {
		t.Fatal("byte cut at 120 should split the trailing CJK rune")
	}

	if got := truncateRunes("x", 0); got != "" {
		t.Fatalf("n=0: got %q", got)
	}
	if got := truncateRunes("x", -1); got != "" {
		t.Fatalf("n<0: got %q", got)
	}
	if got := truncateRunes("", 10); got != "" {
		t.Fatalf("empty: got %q", got)
	}
}
