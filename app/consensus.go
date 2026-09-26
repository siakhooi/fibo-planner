package main

import (
	"sort"
	"strconv"
	"strings"
)

type voteTally struct {
	points string
	count  int
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

func allVotersHaveVoted(rows []participant) bool {
	voters := 0
	for _, row := range rows {
		if row.observer {
			continue
		}
		if row.points == "" {
			return false
		}
		voters++
	}
	return voters > 0
}

func tallyVotes(rows []participant) []voteTally {
	counts := make(map[string]int)
	for _, row := range rows {
		if row.observer || row.points == "" {
			continue
		}
		counts[row.points]++
	}
	out := make([]voteTally, 0, len(counts))
	for points, n := range counts {
		out = append(out, voteTally{points: points, count: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].count != out[j].count {
			return out[i].count > out[j].count
		}
		pi, _ := strconv.Atoi(out[i].points)
		pj, _ := strconv.Atoi(out[j].points)
		return pi < pj
	})
	return out
}

func percentOf(count, total int) int {
	if total <= 0 {
		return 0
	}
	return (count*100 + total/2) / total
}

func voteSpread(rows []participant) int {
	minRank, maxRank := -1, -1
	for _, row := range rows {
		if row.observer || row.points == "" {
			continue
		}
		rank, ok := pointsRank(row.points)
		if !ok {
			continue
		}
		if minRank < 0 || rank < minRank {
			minRank = rank
		}
		if maxRank < 0 || rank > maxRank {
			maxRank = rank
		}
	}
	if minRank < 0 {
		return 0
	}
	return maxRank - minRank
}

func meetsSpread(spread, maxSpread int) bool {
	return spread <= maxSpread
}

func agreedPoints(tallies []voteTally, total, consensus, spread, maxSpread int) string {
	if !meetsSpread(spread, maxSpread) {
		return "N/A"
	}
	matched := make([]string, 0, len(tallies))
	for _, t := range tallies {
		if meetsConsensus(percentOf(t.count, total), consensus) {
			matched = append(matched, t.points)
		}
	}
	if len(matched) == 0 {
		return "N/A"
	}
	return strings.Join(matched, ", ")
}
