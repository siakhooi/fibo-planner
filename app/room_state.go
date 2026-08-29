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

	var listHTML strings.Builder
	listHTML.WriteString(`<table id="user-list" class="user-table" hx-swap-oob="true">`)
	listHTML.WriteString(`<thead><tr><th scope="col">Name</th><th scope="col">Points</th></tr></thead><tbody>`)
	for _, row := range rows {
		fmt.Fprintf(&listHTML, "<tr><td>%s</td><td>%s</td></tr>", html.EscapeString(row.name), html.EscapeString(row.points))
	}
	listHTML.WriteString("</tbody></table>")

	return fmt.Sprintf(
		`<strong id="session-count" hx-swap-oob="true">%d</strong>%s`,
		n, listHTML.String(),
	)
}
