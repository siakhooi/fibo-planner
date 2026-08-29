package main

import (
	"fmt"
	"html"
	"sort"
	"strings"
)

func roomStateHTML(n int, rows []participant, alwaysShow bool) string {
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
		fmt.Fprintf(&listHTML, "<tr><td>%s</td><td%s>%s</td></tr>", html.EscapeString(row.name), flash, html.EscapeString(points))
	}
	listHTML.WriteString("</tbody></table>")

	pressed := "false"
	if alwaysShow {
		pressed = "true"
	}

	return fmt.Sprintf(
		`<strong id="session-count" hx-swap-oob="true">%d</strong>%s`+
			`<button type="submit" id="always-show-votes" hx-swap-oob="true" aria-pressed="%s">Always show votes</button>`,
		n, listHTML.String(), pressed,
	)
}
