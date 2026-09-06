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

const (
	adminAlwaysShowVotes    = "always-show-votes"
	adminResetTopic         = "reset-topic"
	adminObserverMode       = "observer-mode"
	adminConsensusAgreement = "consensus-agreement"
	minConsensusPercent     = 50
	maxConsensusPercent     = 100
	defaultConsensusPercent = 100
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
	case adminAlwaysShowVotes, adminResetTopic, adminObserverMode, adminConsensusAgreement:
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

func parseResetTopic(payload []byte) (title string, clearVotes bool) {
	var m map[string]any
	if err := json.Unmarshal(payload, &m); err != nil {
		return "", false
	}
	return jsonString(m["topic-title"]), jsonTruthy(m["clear-votes"])
}

func jsonString(v any) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
}

func jsonTruthy(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		return s == "on" || s == "true" || s == "yes" || s == "1"
	case float64:
		return t != 0
	default:
		return false
	}
}
