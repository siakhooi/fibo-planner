package main

import (
	"encoding/json"
	"strconv"
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

// expecting something like this
// {"points":"8","HEADERS":{"HX-Request":"true","HX-Current-URL":"..."}}
func parseVotePoints(payload []byte) (string, bool) {
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
