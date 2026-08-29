package main

import (
	"fmt"
	"html"
	"sort"
	"strings"
)

func roomStateHTML(n int, rows []participant) string {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].name != rows[j].name {
			return rows[i].name < rows[j].name
		}
		return rows[i].points < rows[j].points
	})
	masked := false
	for _, row := range rows {
		if row.points == "" {
			masked = true
			break
		}
	}
	var listHTML strings.Builder
	listHTML.WriteString(`<table id="user-list" class="user-table" hx-swap-oob="true">`)
	listHTML.WriteString(`<thead><tr><th scope="col">Name</th><th scope="col">Points</th></tr></thead><tbody>`)
	for _, row := range rows {
		points := row.points
		if masked && points != "" {
			points = "???"
		}
		flash := ""
		if row.flash {
			flash = ` class="vote-flash"`
		}
		fmt.Fprintf(&listHTML, "<tr><td>%s</td><td%s>%s</td></tr>", html.EscapeString(row.name), flash, html.EscapeString(points))
	}
	listHTML.WriteString("</tbody></table>")

	return fmt.Sprintf(
		`<strong id="session-count" hx-swap-oob="true">%d</strong>%s`,
		n, listHTML.String(),
	)
}
