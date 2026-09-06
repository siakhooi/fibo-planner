package main

import (
	"fmt"
	"html"
	"sort"
	"strconv"
	"strings"
)

type voteTally struct {
	points string
	count  int
}

func roomStateHTML(n int, rows []participant, alwaysShow bool, topic string, consensus int) string {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].observer != rows[j].observer {
			return !rows[i].observer
		}
		if rows[i].name != rows[j].name {
			return rows[i].name < rows[j].name
		}
		return rows[i].points < rows[j].points
	})
	masked := false
	if !alwaysShow {
		for _, row := range rows {
			if row.observer {
				continue
			}
			if row.points == "" {
				masked = true
				break
			}
		}
	}
	var listHTML strings.Builder
	listHTML.WriteString(`<table id="user-list" class="user-table" hx-swap-oob="true">`)
	listHTML.WriteString(`<thead><tr><th scope="col">Name</th><th scope="col">Points</th></tr></thead><tbody>`)
	for _, row := range rows {
		points := row.points
		if row.observer {
			points = "observer"
		} else if masked && points != "" {
			points = "???"
		}
		flash := ""
		if row.flash {
			flash = ` class="vote-flash"`
		}
		fmt.Fprintf(&listHTML, "<tr><td%s>%s</td><td%s>%s</td></tr>", flash, html.EscapeString(row.name), flash, html.EscapeString(points))
	}
	listHTML.WriteString("</tbody></table>")

	pressed := "false"
	if alwaysShow {
		pressed = "true"
	}

	return fmt.Sprintf(
		`<strong id="session-count" hx-swap-oob="true">%d</strong>`+
			"%s"+
			`<button type="submit" id="always-show-votes" hx-swap-oob="true" aria-pressed="%s">Always show votes</button>`+
			"%s"+
			"%s"+
			"%s",
		n,
		listHTML.String(),
		pressed,
		consensusControlsHTML(consensus),
		voteResultsHTML(rows, consensus),
		topicHeadingHTML(topic),
	)
}

func consensusControlsHTML(percent int) string {
	percent = normalizeConsensusPercent(percent)
	return fmt.Sprintf(
		`<div id="consensus-controls" hx-swap-oob="true">`+
			`<label for="consensus-percent">Percentage <output id="consensus-percent-value" for="consensus-percent">%d</output></label>`+
			`<input type="range" id="consensus-percent" name="percentage" min="%d" max="%d" step="1" value="%d" list="consensus-majors" />`+
			`<div class="consensus-majors" aria-hidden="true"><span>50</span><span>60</span><span>70</span><span>80</span><span>90</span><span>100</span></div>`+
			`<datalist id="consensus-majors">`+
			`<option value="50"></option><option value="60"></option><option value="70"></option>`+
			`<option value="80"></option><option value="90"></option><option value="100"></option>`+
			`</datalist>`+
			`</div>`,
		percent,
		minConsensusPercent,
		maxConsensusPercent,
		percent,
	)
}

func topicHeadingHTML(topic string) string {
	if topic == "" {
		return `<h2 id="topic-title" class="topic-title" hx-swap-oob="true" hidden></h2>`
	}
	return fmt.Sprintf(`<h2 id="topic-title" class="topic-title" hx-swap-oob="true">%s</h2>`, html.EscapeString(topic))
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

const voteResultsHead = `<thead><tr><th scope="col">Points</th><th scope="col">Count</th><th scope="col">%</th></tr></thead>`

func percentOf(count, total int) int {
	if total <= 0 {
		return 0
	}
	return (count*100 + total/2) / total
}

func agreedPoints(tallies []voteTally, total, consensus int) string {
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

func agreedPointsHTML(points string) string {
	if points == "" {
		return `<p id="agreed-points" hx-swap-oob="true" hidden></p>`
	}
	kind := "agreed-yes"
	if points == "N/A" {
		kind = "agreed-no"
	}
	return fmt.Sprintf(
		`<p id="agreed-points" class="%s" hx-swap-oob="true">Agreed Points: <strong>%s</strong></p>`,
		kind,
		html.EscapeString(points),
	)
}

func voteResultsHTML(rows []participant, consensus int) string {
	var b strings.Builder
	if !allVotersHaveVoted(rows) {
		b.WriteString(agreedPointsHTML(""))
		b.WriteString(`<table id="vote-results" class="user-table results-table" hx-swap-oob="true" hidden>`)
		b.WriteString(voteResultsHead)
		b.WriteString("<tbody></tbody></table>")
		return b.String()
	}

	tallies := tallyVotes(rows)
	total := 0
	maxCount := 0
	if len(tallies) > 0 {
		maxCount = tallies[0].count
	}
	for _, t := range tallies {
		total += t.count
	}

	b.WriteString(agreedPointsHTML(agreedPoints(tallies, total, consensus)))
	b.WriteString(`<table id="vote-results" class="user-table results-table" hx-swap-oob="true">`)
	b.WriteString(voteResultsHead)
	b.WriteString("<tbody>")
	for _, t := range tallies {
		cls := ""
		if t.count == maxCount {
			cls = ` class="vote-leader"`
		}
		fmt.Fprintf(&b, "<tr%s><td>%s</td><td>%d</td><td>%d%%</td></tr>", cls, html.EscapeString(t.points), t.count, percentOf(t.count, total))
	}
	b.WriteString("</tbody></table>")
	return b.String()
}
