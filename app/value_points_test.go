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

func TestParseTopicTitle(t *testing.T) {
	t.Parallel()

	if got := parseTopicTitle([]byte(`{"admin":"set-topic","topic-title":" Login "}`)); got != "Login" {
		t.Fatalf("got title=%q", got)
	}
	if got := parseTopicTitle([]byte(`{"admin":"set-topic"}`)); got != "" {
		t.Fatalf("missing title should be empty, got %q", got)
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
