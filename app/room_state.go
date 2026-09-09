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

func roomStateHTML(n int, rows []participant, alwaysShow bool, topic string, consensus, maxSpread int) string {
	return renderRoomState(n, rows, alwaysShow, topic, consensus, maxSpread, nil)
}

func renderRoomState(n int, rows []participant, alwaysShow bool, topic string, consensus, maxSpread int, preloaded []string) string {
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
			"%s"+
			"%s"+
			"%s",
		n,
		listHTML.String(),
		pressed,
		consensusControlsHTML(consensus, maxSpread),
		voteResultsHTML(rows, consensus, maxSpread),
		topicHeadingHTML(topic),
		loadNextTopicButtonHTML(preloaded),
		preloadedTopicsDataHTML(preloaded),
	)
}

type maturityPreset struct {
	label   string
	percent int
	spread  int
}

var teamMaturityPresets = []maturityPreset{
	{label: "full (100%, 0 spread)", percent: 100, spread: 0},
	{label: "good (80%, 1 spread)", percent: 80, spread: 1},
	{label: "relaxed (50%, 3 spreads)", percent: 50, spread: 3},
}

func consensusControlsHTML(percent, maxSpread int) string {
	percent = normalizeConsensusPercent(percent)
	maxSpread = normalizeMaxSpread(maxSpread)
	return fmt.Sprintf(
		`<div id="consensus-controls" hx-swap-oob="true">`+
			`<div class="consensus-slider">`+
			`<label for="consensus-percent">Percentage <output id="consensus-percent-value" for="consensus-percent">%d</output></label>`+
			`<input type="range" id="consensus-percent" name="percentage" min="%d" max="%d" step="1" value="%d" list="consensus-majors" />`+
			`<div class="consensus-majors" aria-hidden="true"><span>50</span><span>60</span><span>70</span><span>80</span><span>90</span><span>100</span></div>`+
			`<datalist id="consensus-majors">`+
			`<option value="50"></option><option value="60"></option><option value="70"></option>`+
			`<option value="80"></option><option value="90"></option><option value="100"></option>`+
			`</datalist>`+
			`</div>`+
			`<div class="consensus-slider">`+
			`<label for="consensus-max-spread">Max Spread <output id="consensus-max-spread-value" for="consensus-max-spread">%d</output></label>`+
			`<input type="range" id="consensus-max-spread" name="max-spread" min="%d" max="%d" step="1" value="%d" list="consensus-spread-ticks" />`+
			`<div class="consensus-majors" aria-hidden="true"><span>0</span><span>1</span><span>2</span><span>3</span><span>4</span><span>5</span><span>6</span></div>`+
			`<datalist id="consensus-spread-ticks">`+
			`<option value="0"></option><option value="1"></option><option value="2"></option>`+
			`<option value="3"></option><option value="4"></option><option value="5"></option>`+
			`<option value="6"></option>`+
			`</datalist>`+
			`</div>`+
			"%s"+
			`</div>`,
		percent,
		minConsensusPercent,
		maxConsensusPercent,
		percent,
		maxSpread,
		minMaxSpread,
		maxMaxSpread,
		maxSpread,
		maturityPresetsHTML(percent, maxSpread),
	)
}

func maturityPresetsHTML(percent, maxSpread int) string {
	var b strings.Builder
	b.WriteString(`<div class="maturity-presets">`)
	b.WriteString(`<h4 id="maturity-presets-heading">Team Maturity(presets)</h4>`)
	b.WriteString(`<ul aria-labelledby="maturity-presets-heading">`)
	for _, p := range teamMaturityPresets {
		pressed := "false"
		if p.percent == percent && p.spread == maxSpread {
			pressed = "true"
		}
		fmt.Fprintf(
			&b,
			`<li><button type="button" class="maturity-preset" data-percentage="%d" data-max-spread="%d" aria-pressed="%s">%s</button></li>`,
			p.percent,
			p.spread,
			pressed,
			html.EscapeString(p.label),
		)
	}
	b.WriteString(`</ul></div>`)
	return b.String()
}

func topicHeadingHTML(topic string) string {
	if topic == "" {
		return `<h2 id="topic-title" class="topic-title" hx-swap-oob="true" hidden></h2>`
	}
	return fmt.Sprintf(`<h2 id="topic-title" class="topic-title" hx-swap-oob="true">%s</h2>`, html.EscapeString(topic))
}

func loadNextTopicButtonHTML(topics []string) string {
	if len(topics) == 0 {
		return `<button type="submit" id="load-next-topic" hx-swap-oob="true" disabled>Load Next Topic</button>`
	}
	return fmt.Sprintf(
		`<button type="submit" id="load-next-topic" hx-swap-oob="true" title="Next Topic: %s">Load Next Topic [%d]</button>`,
		html.EscapeString(topics[0]),
		len(topics),
	)
}

func preloadedTopicsDataHTML(topics []string) string {
	return fmt.Sprintf(
		`<pre id="preloaded-topics-data" hx-swap-oob="true" hidden>%s</pre>`,
		html.EscapeString(strings.Join(topics, "\n")),
	)
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

func percentRequireLabel(threshold int) string {
	if threshold <= minConsensusPercent {
		return "require >50%"
	}
	return fmt.Sprintf("require >=%d%%", threshold)
}

func spreadRequireLabel(maxSpread int) string {
	if maxSpread == 0 {
		return "require=0"
	}
	return fmt.Sprintf("require <=%d", maxSpread)
}

func agreementMark(ok bool) string {
	if ok {
		return "✓"
	}
	return "X"
}

func agreementKind(ok bool) string {
	if ok {
		return "met"
	}
	return "unmet"
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

func agreementStatusHTML(show bool, leadingPercent, consensus, spread, maxSpread int) string {
	if !show {
		return `<p id="agreement-status" class="agreement-status" hx-swap-oob="true" hidden></p>`
	}
	percentOK := meetsConsensus(leadingPercent, consensus)
	spreadOK := meetsSpread(spread, maxSpread)
	return fmt.Sprintf(
		`<p id="agreement-status" class="agreement-status" hx-swap-oob="true">`+
			`<span class="%s">%s %d%% (%s)</span>`+
			`<span class="%s">%s spread = %d (%s)</span>`+
			`</p>`,
		agreementKind(percentOK),
		agreementMark(percentOK),
		leadingPercent,
		percentRequireLabel(consensus),
		agreementKind(spreadOK),
		agreementMark(spreadOK),
		spread,
		spreadRequireLabel(maxSpread),
	)
}

func voteResultsHTML(rows []participant, consensus, maxSpread int) string {
	var b strings.Builder
	if !allVotersHaveVoted(rows) {
		b.WriteString(agreedPointsHTML(""))
		b.WriteString(agreementStatusHTML(false, 0, consensus, 0, maxSpread))
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
	leadingPercent := 0
	if total > 0 && len(tallies) > 0 {
		leadingPercent = percentOf(tallies[0].count, total)
	}
	spread := voteSpread(rows)

	b.WriteString(agreedPointsHTML(agreedPoints(tallies, total, consensus, spread, maxSpread)))
	b.WriteString(agreementStatusHTML(true, leadingPercent, consensus, spread, maxSpread))
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
