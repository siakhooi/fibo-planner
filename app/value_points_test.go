package main

import (
	"testing"
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
		{name: "admin is not a vote", payload: `{"admin":"reset-topic"}`, ok: false},
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
		{name: "reset topic", payload: `{"admin":"reset-topic","HEADERS":{}}`, want: adminResetTopic, ok: true},
		{name: "always show", payload: `{"admin":"always-show-votes"}`, want: adminAlwaysShowVotes, ok: true},
		{name: "observer", payload: `{"admin":"observer-mode"}`, want: adminObserverMode, ok: true},
		{name: "consensus", payload: `{"admin":"consensus-agreement","percentage":"75"}`, want: adminConsensusAgreement, ok: true},
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

func TestParseResetTopic(t *testing.T) {
	t.Parallel()

	title, clear := parseResetTopic([]byte(`{"admin":"reset-topic","topic-title":" Login ","clear-votes":"on"}`))
	if title != "Login" || !clear {
		t.Fatalf("got title=%q clear=%v", title, clear)
	}

	title, clear = parseResetTopic([]byte(`{"admin":"reset-topic","topic-title":"Keep votes"}`))
	if title != "Keep votes" || clear {
		t.Fatalf("unchecked should not clear: title=%q clear=%v", title, clear)
	}

	title, clear = parseResetTopic([]byte(`{"admin":"reset-topic","topic-title":"X","clear-votes":false}`))
	if title != "X" || clear {
		t.Fatalf("false should not clear: title=%q clear=%v", title, clear)
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
