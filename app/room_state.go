package main

import (
	"bytes"
	"fmt"
	htmltemplate "html/template"
	"log"
	"sort"
	"strings"
)

func renderRoomState(n int, rows []participant, alwaysShow bool, topic string, consensus, maxSpread int, preloaded []string) string {
	sortParticipants(rows)
	return execRoomTemplate("room-state", roomStateData{
		Count:           n,
		Users:           roomUserViews(rows, maskedVotes(alwaysShow, rows)),
		AlwaysPressed:   pressedAttr(alwaysShow),
		ObserverPressed: pressedAttr(selfIsObserver(rows)),
		Consensus:       consensusViewFrom(consensus, maxSpread),
		Results:         voteResultsViewFrom(rows, consensus, maxSpread),
		Topic:           topicView{Title: topic, Empty: topic == ""},
		Queue:           queueViewFrom(preloaded),
	})
}

type roomStateData struct {
	Count           int
	Users           []roomUserView
	AlwaysPressed   string
	ObserverPressed string
	Consensus       consensusView
	Results         voteResultsView
	Topic           topicView
	Queue           queueView
}

type roomUserView struct {
	Name   string
	Points string
	Self   bool
	Flash  bool
}

type topicView struct {
	Title string
	Empty bool
}

type queueView struct {
	Empty bool
	Next  string
	Count int
	Body  string
}

type maturityButton struct {
	Label   string
	Percent int
	Spread  int
	Pressed string
}

type consensusView struct {
	Percent        int
	MinPercent     int
	MaxPercent     int
	MaxSpread      int
	MinSpread      int
	MaxSpreadLimit int
	PercentTicks   []int
	SpreadTicks    []int
	Presets        []maturityButton
}

// roomPageData is the first paint of a room. The widgets it embeds are the
// same templates the WebSocket later swaps in.
type roomPageData struct {
	RoomID          string
	RoomName        string
	Count           int
	AlwaysPressed   string
	ObserverPressed string
	Users           []roomUserView
	Consensus       consensusView
	Results         voteResultsView
	Topic           topicView
	Queue           queueView
}

type voteResultRow struct {
	Leader  bool
	Points  string
	Count   int
	Percent int
}

type voteResultsView struct {
	Hidden         bool
	ShowAgreed     bool
	AgreedClass    string
	AgreedPoints   string
	ShowAgreement  bool
	PercentKind    string
	PercentMark    string
	LeadingPercent int
	PercentRequire htmltemplate.HTML
	SpreadKind     string
	SpreadMark     string
	Spread         int
	SpreadRequire  htmltemplate.HTML
	Rows           []voteResultRow
}

func execRoomTemplate(name string, data any) string {
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		log.Printf("room state template %s: %v", name, err)
		return ""
	}
	return buf.String()
}

func sortParticipants(rows []participant) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].observer != rows[j].observer {
			return !rows[i].observer
		}
		if rows[i].name != rows[j].name {
			return rows[i].name < rows[j].name
		}
		return rows[i].points < rows[j].points
	})
}

func maskedVotes(alwaysShow bool, rows []participant) bool {
	if alwaysShow {
		return false
	}
	for _, row := range rows {
		if row.observer {
			continue
		}
		if row.points == "" {
			return true
		}
	}
	return false
}

func selfIsObserver(rows []participant) bool {
	for _, row := range rows {
		if row.self && row.observer {
			return true
		}
	}
	return false
}

func pressedAttr(on bool) string {
	if on {
		return "true"
	}
	return "false"
}

func roomUserViews(rows []participant, masked bool) []roomUserView {
	out := make([]roomUserView, len(rows))
	for i, row := range rows {
		points := row.points
		if row.observer {
			points = "observer"
		} else if masked && points != "" {
			points = "???"
		}
		out[i] = roomUserView{
			Name:   row.name,
			Points: points,
			Self:   row.self,
			Flash:  row.flash,
		}
	}
	return out
}

func queueViewFrom(topics []string) queueView {
	if len(topics) == 0 {
		return queueView{Empty: true}
	}
	return queueView{
		Next:  topics[0],
		Count: len(topics),
		Body:  strings.Join(topics, "\n"),
	}
}

type maturityPreset struct {
	label   string
	percent int
	spread  int
}

var teamMaturityPresets = []maturityPreset{
	{label: "full (100%, 0 spread)", percent: 100, spread: 0},
	{label: "good (80%, 1 spread)", percent: 80, spread: 1},
	{label: "relaxed (50%, spread of 3)", percent: 50, spread: 3},
}

func roomPageFrom(roomID string, h *Hub) roomPageData {
	name, count, alwaysShow, topic, consensus, maxSpread, preloaded := h.pageView()
	return roomPageData{
		RoomID:          roomID,
		RoomName:        name,
		Count:           count,
		AlwaysPressed:   pressedAttr(alwaysShow),
		ObserverPressed: pressedAttr(false),
		Consensus:       consensusViewFrom(consensus, maxSpread),
		Results:         voteResultsViewFrom(nil, consensus, maxSpread),
		Topic:           topicView{Title: topic, Empty: topic == ""},
		Queue:           queueViewFrom(preloaded),
	}
}

func consensusViewFrom(percent, maxSpread int) consensusView {
	percent = normalizeConsensusPercent(percent)
	maxSpread = normalizeMaxSpread(maxSpread)
	return consensusView{
		Percent:        percent,
		MinPercent:     minConsensusPercent,
		MaxPercent:     maxConsensusPercent,
		MaxSpread:      maxSpread,
		MinSpread:      minMaxSpread,
		MaxSpreadLimit: maxMaxSpread,
		PercentTicks:   percentTicks(),
		SpreadTicks:    spreadTicks(),
		Presets:        maturityButtons(percent, maxSpread),
	}
}

func percentTicks() []int {
	return percentTicksForStep(consensusPercentTickStep)
}

func percentTicksForStep(step int) []int {
	if step <= 0 {
		step = 1
	}
	ticks := make([]int, 0, (maxConsensusPercent-minConsensusPercent)/step+1)
	for n := minConsensusPercent; n <= maxConsensusPercent; n += step {
		ticks = append(ticks, n)
	}
	if len(ticks) == 0 || ticks[len(ticks)-1] != maxConsensusPercent {
		ticks = append(ticks, maxConsensusPercent)
	}
	return ticks
}

func spreadTicks() []int {
	ticks := make([]int, 0, maxMaxSpread-minMaxSpread+1)
	for i := minMaxSpread; i <= maxMaxSpread; i++ {
		ticks = append(ticks, i)
	}
	return ticks
}

func maturityButtons(percent, maxSpread int) []maturityButton {
	out := make([]maturityButton, len(teamMaturityPresets))
	for i, p := range teamMaturityPresets {
		out[i] = maturityButton{
			Label:   p.label,
			Percent: p.percent,
			Spread:  p.spread,
			Pressed: pressedAttr(p.percent == percent && p.spread == maxSpread),
		}
	}
	return out
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

func voteResultsViewFrom(rows []participant, consensus, maxSpread int) voteResultsView {
	if !allVotersHaveVoted(rows) {
		return voteResultsView{Hidden: true}
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
	points := agreedPoints(tallies, total, consensus, spread, maxSpread)
	kind := "agreed-yes"
	if points == "N/A" {
		kind = "agreed-no"
	}
	percentOK := meetsConsensus(leadingPercent, consensus)
	spreadOK := meetsSpread(spread, maxSpread)
	rowsOut := make([]voteResultRow, len(tallies))
	for i, t := range tallies {
		rowsOut[i] = voteResultRow{
			Leader:  t.count == maxCount,
			Points:  t.points,
			Count:   t.count,
			Percent: percentOf(t.count, total),
		}
	}
	return voteResultsView{
		ShowAgreed:     true,
		AgreedClass:    kind,
		AgreedPoints:   points,
		ShowAgreement:  true,
		PercentKind:    agreementKind(percentOK),
		PercentMark:    agreementMark(percentOK),
		LeadingPercent: leadingPercent,
		PercentRequire: htmltemplate.HTML(percentRequireLabel(consensus)),
		SpreadKind:     agreementKind(spreadOK),
		SpreadMark:     agreementMark(spreadOK),
		Spread:         spread,
		SpreadRequire:  htmltemplate.HTML(spreadRequireLabel(maxSpread)),
		Rows:           rowsOut,
	}
}
