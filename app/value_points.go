package main

import (
	"encoding/json"
	"strconv"
	"strings"
)

var allowedVotePoints = map[string]bool{
	"":   true,
	"1":  true,
	"2":  true,
	"3":  true,
	"5":  true,
	"8":  true,
	"13": true,
	"20": true,
}

// voteScale is Fibonacci story points in rank order. Spread is the rank
// distance between the lowest and highest votes (3 and 5 → 1, 1 and 20 → 6).
var voteScale = []string{"1", "2", "3", "5", "8", "13", "20"}

const (
	adminAlwaysShowVotes    = "always-show-votes"
	adminClearVotes         = "clear-votes"
	adminSetTopic           = "set-topic"
	adminObserverMode       = "observer-mode"
	adminConsensusAgreement = "consensus-agreement"
	adminLoadNextTopic      = "load-next-topic"
	adminSetPreloadedTopics = "set-preloaded-topics"
	minConsensusPercent     = 50
	maxConsensusPercent     = 100
	defaultConsensusPercent = 100
	minMaxSpread            = 0
	maxMaxSpread            = 6
	defaultMaxSpread        = 0
	maxTopicTitleLen        = 120
	maxPreloadedTopicCount  = 200
)

func parseVotePoints(payload []byte) (string, bool) {
	// expecting something like this
	// {"points":"8","HEADERS":{"HX-Request":"true","HX-Current-URL":"..."}}

	var m map[string]any
	if err := json.Unmarshal(payload, &m); err != nil {
		return "", false
	}
	v, ok := m["points"]
	if !ok {
		return "", false
	}
	var s string
	switch t := v.(type) {
	case string:
		s = t
	case float64:
		s = strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return "", false
	}
	if !allowedVotePoints[s] {
		return "", false
	}

	return s, true

}

func parseAdminAction(payload []byte) (string, bool) {
	var m map[string]any
	if err := json.Unmarshal(payload, &m); err != nil {
		return "", false
	}
	v, ok := m["admin"]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	if !ok {
		return "", false
	}
	switch s {
	case adminAlwaysShowVotes, adminClearVotes, adminSetTopic, adminObserverMode, adminConsensusAgreement, adminLoadNextTopic, adminSetPreloadedTopics:
		return s, true
	default:
		return "", false
	}

}

func parseConsensusPercent(payload []byte) (int, bool) {
	var m map[string]any
	if err := json.Unmarshal(payload, &m); err != nil {
		return 0, false
	}
	n, ok := jsonInt(m["percentage"])
	if !ok || n < minConsensusPercent || n > maxConsensusPercent {
		return 0, false
	}
	return n, true
}

func parseMaxSpread(payload []byte) (int, bool) {
	var m map[string]any
	if err := json.Unmarshal(payload, &m); err != nil {
		return 0, false
	}
	n, ok := jsonInt(m["max-spread"])
	if !ok || n < minMaxSpread || n > maxMaxSpread {
		return 0, false
	}
	return n, true
}

func jsonInt(v any) (int, bool) {
	switch t := v.(type) {
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(t))
		if err != nil {
			return 0, false
		}
		return n, true
	case float64:
		n := int(t)
		if float64(n) != t {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func normalizeConsensusPercent(n int) int {
	if n < minConsensusPercent || n > maxConsensusPercent {
		return defaultConsensusPercent
	}
	return n
}

func normalizeMaxSpread(n int) int {
	if n < minMaxSpread || n > maxMaxSpread {
		return defaultMaxSpread
	}
	return n
}

func pointsRank(points string) (int, bool) {
	for i, p := range voteScale {
		if p == points {
			return i, true
		}
	}
	return 0, false
}

func meetsConsensus(percent, threshold int) bool {
	switch {
	case threshold <= minConsensusPercent:
		return percent > minConsensusPercent
	case threshold >= maxConsensusPercent:
		return percent == maxConsensusPercent
	default:
		return percent >= threshold
	}
}

func parseTopicTitle(payload []byte) string {
	var m map[string]any
	if err := json.Unmarshal(payload, &m); err != nil {
		return ""
	}
	return jsonString(m["topic-title"])
}

func parsePreloadedTopics(payload []byte) []string {
	var m map[string]any
	if err := json.Unmarshal(payload, &m); err != nil {
		return nil
	}
	return normalizePreloadedTopics(jsonString(m["preloaded-topics"]))
}

func normalizePreloadedTopics(raw string) []string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	lines := strings.Split(raw, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len(line) > maxTopicTitleLen {
			line = line[:maxTopicTitleLen]
		}
		out = append(out, line)
		if len(out) >= maxPreloadedTopicCount {
			break
		}
	}
	return out
}

func jsonString(v any) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}
