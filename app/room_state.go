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

func roomStateHTML(n int, rows []participant, alwaysShow bool, topic string) string {
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
			"%s",
		n,
		listHTML.String(),
		pressed,
		voteResultsHTML(rows),
		topicHeadingHTML(topic),
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

func voteResultsHTML(rows []participant) string {
	var b strings.Builder
	if !allVotersHaveVoted(rows) {
		b.WriteString(`<table id="vote-results" class="user-table results-table" hx-swap-oob="true" hidden>`)
		b.WriteString(voteResultsHead)
		b.WriteString("<tbody></tbody></table>")
	} else {
		b.WriteString(`<table id="vote-results" class="user-table results-table" hx-swap-oob="true">`)
		b.WriteString(voteResultsHead)
		b.WriteString("<tbody>")

		tallies := tallyVotes(rows)
		total := 0
		maxCount := 0
		if len(tallies) > 0 {
			maxCount = tallies[0].count
		}
		for _, t := range tallies {
			total += t.count
		}
		for _, t := range tallies {
			cls := ""
			if t.count == maxCount {
				cls = ` class="vote-leader"`
			}
			fmt.Fprintf(&b, "<tr%s><td>%s</td><td>%d</td><td>%d%%</td></tr>", cls, html.EscapeString(t.points), t.count, percentOf(t.count, total))
		}

		b.WriteString("</tbody></table>")
	}
	return b.String()
}
