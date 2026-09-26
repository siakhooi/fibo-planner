package main

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// voteScale is Fibonacci story points in rank order. Spread is the rank
// distance between the lowest and highest votes (3 and 5 → 1, 1 and 20 → 6).
// Card buttons on the room page are generated from this slice.
var voteScale = []string{"1", "2", "3", "5", "8", "13", "20"}

// allowedVotePoints is the blank (clear) vote plus every value in voteScale.
var allowedVotePoints = func() map[string]bool {
	m := map[string]bool{"": true}
	for _, p := range voteScale {
		m[p] = true
	}
	return m
}()

// maxMaxSpread is the rank distance from the first to last card on voteScale.
var maxMaxSpread = len(voteScale) - 1

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
	defaultMaxSpread        = 0
	maxDisplayNameLen       = 120
	maxTopicTitleLen        = maxDisplayNameLen
	maxPreloadedTopicCount  = 200

	// roomIdleEvictionDelay is how long a room with zero WebSocket connections may stay before it is removed.
	// README documents this as "30 minutes".
	roomIdleEvictionDelay = 30 * time.Minute
)

// truncateRunes returns the first n runes of s. n <= 0 yields "".
func truncateRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}

// wsMessage is one room WebSocket JSON body (join, vote, or admin).
// HTMX may also send a HEADERS object; it is ignored.
type wsMessage struct {
	Admin           any `json:"admin"`
	Name            any `json:"name"`
	Points          any `json:"points"`
	TopicTitle      any `json:"topic-title"`
	Percentage      any `json:"percentage"`
	MaxSpread       any `json:"max-spread"`
	PreloadedTopics any `json:"preloaded-topics"`
}

func parseWSMessage(payload []byte) (wsMessage, bool) {
	var m wsMessage
	if err := json.Unmarshal(payload, &m); err != nil {
		return wsMessage{}, false
	}
	return m, true
}

func (m wsMessage) votePoints() (string, bool) {
	// expecting something like this
	// {"points":"8","HEADERS":{"HX-Request":"true","HX-Current-URL":"..."}}
	if m.Points == nil {
		return "", false
	}
	var s string
	switch t := m.Points.(type) {
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

func (m wsMessage) adminAction() (string, bool) {
	s, ok := m.Admin.(string)
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

func (m wsMessage) joinName() (string, bool) {
	if m.Name == nil {
		return "", false
	}
	s, ok := m.Name.(string)
	if !ok {
		return "", false
	}
	return strings.TrimSpace(s), true
}

func (m wsMessage) consensusPercent() (int, bool) {
	n, ok := jsonInt(m.Percentage)
	if !ok || n < minConsensusPercent || n > maxConsensusPercent {
		return 0, false
	}
	return n, true
}

func (m wsMessage) maxSpread() (int, bool) {
	n, ok := jsonInt(m.MaxSpread)
	if !ok || n < minMaxSpread || n > maxMaxSpread {
		return 0, false
	}
	return n, true
}

func (m wsMessage) topicTitle() string {
	return jsonString(m.TopicTitle)
}

func (m wsMessage) preloadedTopics() []string {
	return normalizePreloadedTopics(jsonString(m.PreloadedTopics))
}

// Thin payload wrappers keep unit tests focused on wire formats.
func parseVotePoints(payload []byte) (string, bool) {
	m, ok := parseWSMessage(payload)
	if !ok {
		return "", false
	}
	return m.votePoints()
}

func parseAdminAction(payload []byte) (string, bool) {
	m, ok := parseWSMessage(payload)
	if !ok {
		return "", false
	}
	return m.adminAction()
}

func parseJoinName(payload []byte) (string, bool) {
	m, ok := parseWSMessage(payload)
	if !ok {
		return "", false
	}
	return m.joinName()
}

func parseConsensusPercent(payload []byte) (int, bool) {
	m, ok := parseWSMessage(payload)
	if !ok {
		return 0, false
	}
	return m.consensusPercent()
}

func parseMaxSpread(payload []byte) (int, bool) {
	m, ok := parseWSMessage(payload)
	if !ok {
		return 0, false
	}
	return m.maxSpread()
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
	m, ok := parseWSMessage(payload)
	if !ok {
		return ""
	}
	return m.topicTitle()
}

func parsePreloadedTopics(payload []byte) []string {
	m, ok := parseWSMessage(payload)
	if !ok {
		return nil
	}
	return m.preloadedTopics()
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
		line = truncateRunes(line, maxTopicTitleLen)
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
