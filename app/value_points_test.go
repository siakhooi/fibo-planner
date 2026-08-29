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