package main

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestParseAppTemplatesNoCustomDir(t *testing.T) {
	t.Parallel()

	tmpl, err := parseAppTemplates(tplFS, "")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	assertStockPages(t, tmpl)
}

func TestParseAppTemplatesMissingDir(t *testing.T) {
	t.Parallel()

	tmpl, err := parseAppTemplates(tplFS, filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	assertStockPages(t, tmpl)
}

func TestParseAppTemplatesNotADirectory(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := parseAppTemplates(tplFS, path)
	if err == nil {
		t.Fatal("expected error for non-directory custom HTML path")
	}
}

func TestParseAppTemplatesAllSnippetsOnAllPages(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeSnippet(t, dir, customHeadFile, `<!--HEAD-MARK--><script>window.__fiboHead=1</script>`)
	writeSnippet(t, dir, customBodyStartFile, `<span id="BODY-START-MARK"></span>`)
	writeSnippet(t, dir, customBodyEndFile, `<span id="BODY-END-MARK"></span>`)

	tmpl, err := parseAppTemplates(tplFS, dir)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	for _, name := range []string{"index.html", "room.html", "room_not_found.html"} {
		page := executeNamed(t, tmpl, name)
		assertSnippetPositions(t, name, page,
			"<!--HEAD-MARK-->",
			`<span id="BODY-START-MARK"></span>`,
			`<span id="BODY-END-MARK"></span>`,
		)
		if !strings.Contains(page, "<script>window.__fiboHead=1</script>") {
			t.Errorf("%s: script in head.html was escaped or dropped", name)
		}
	}
}

func TestParseAppTemplatesOnlyHeadFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeSnippet(t, dir, customHeadFile, `<!--HEAD-ONLY-->`)

	tmpl, err := parseAppTemplates(tplFS, dir)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	page := executeNamed(t, tmpl, "index.html")
	if !strings.Contains(page, "<!--HEAD-ONLY-->") {
		t.Fatal("missing head snippet")
	}
	if strings.Contains(page, "BODY-START-MARK") || strings.Contains(page, "BODY-END-MARK") {
		t.Fatal("unexpected body snippets")
	}
	assertSnippetPositions(t, "index.html", page, "<!--HEAD-ONLY-->", "", "")
}

func TestParseAppTemplatesSnippetTemplateActionsAreLiteral(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeSnippet(t, dir, customHeadFile, `{{.RoomID}}`)

	tmpl, err := parseAppTemplates(tplFS, dir)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	page := executeNamed(t, tmpl, "room.html")
	if !strings.Contains(page, "{{.RoomID}}") {
		t.Fatalf("expected literal {{.RoomID}} in snippet, got:\n%s", page)
	}
	head := page[:indexFold(page, "</head>")]
	if strings.Count(head, "{{.RoomID}}") != 1 {
		t.Fatalf("snippet action should appear once in <head>, got %q", head)
	}
}

func TestApplyCustomHTMLSlotsInsertsAtHeadAndBody(t *testing.T) {
	t.Parallel()

	src := "<!DOCTYPE html><HTML><HEAD><title>t</title></HEAD><BODY class=\"x\"><p>hi</p></BODY></HTML>"
	got := applyCustomHTMLSlots(src)
	assertSnippetPositions(t, "mini", got, "{{customHead}}", "{{customBodyStart}}", "{{customBodyEnd}}")
}

func TestParseAppTemplatesMinimalFS(t *testing.T) {
	t.Parallel()

	htmlFS := fstest.MapFS{
		"page.html": &fstest.MapFile{Data: []byte(`<html><head><title>t</title></head><body><p>hi</p></body></html>`)},
	}
	dir := t.TempDir()
	writeSnippet(t, dir, customBodyEndFile, `<span id="END-MARK"></span>`)

	tmpl, err := parseAppTemplates(htmlFS, dir)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "page.html", nil); err != nil {
		t.Fatalf("execute: %v", err)
	}
	assertSnippetPositions(t, "page.html", buf.String(), "", "", `<span id="END-MARK"></span>`)
}

func writeSnippet(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func executeNamed(t *testing.T, tmpl *template.Template, name string) string {
	t.Helper()
	var buf bytes.Buffer
	var data any
	switch name {
	case "index.html":
		data = struct {
			LobbyCount int
			Rooms      []LobbyRoomRow
		}{}
	case "room.html":
		data = struct {
			RoomID                string
			RoomName              string
			TopicTitle            string
			Count                 int
			ConsensusControlsHTML template.HTML
		}{
			RoomID: "123456",
		}
	case "room_not_found.html":
		data = struct{ RoomID string }{RoomID: "123456"}
	default:
		t.Fatalf("unknown template %s", name)
	}
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		t.Fatalf("execute %s: %v", name, err)
	}
	return buf.String()
}

func assertStockPages(t *testing.T, tmpl *template.Template) {
	t.Helper()
	index := executeNamed(t, tmpl, "index.html")
	if !strings.Contains(index, "<title>Fibo Planner</title>") {
		t.Fatal("index.html missing title")
	}
	room := executeNamed(t, tmpl, "room.html")
	if !strings.Contains(room, "Room 123456") {
		t.Fatal("room.html missing room id")
	}
	missing := executeNamed(t, tmpl, "room_not_found.html")
	if !strings.Contains(missing, "Room does not exist") {
		t.Fatal("room_not_found.html missing copy")
	}
}

func assertSnippetPositions(t *testing.T, name, page, head, bodyStart, bodyEnd string) {
	t.Helper()
	headIdx := indexFold(page, "</head>")
	if headIdx < 0 {
		t.Fatalf("%s: missing </head>", name)
	}
	bodyOpen := reOpenBody.FindStringIndex(page)
	if bodyOpen == nil {
		t.Fatalf("%s: missing <body>", name)
	}
	bodyClose := lastIndexFold(page, "</body>")
	if bodyClose < 0 {
		t.Fatalf("%s: missing </body>", name)
	}

	if head != "" {
		if !strings.Contains(page[:headIdx], head) {
			t.Errorf("%s: %q not in <head>", name, head)
		}
		if strings.Contains(page[headIdx:], head) {
			t.Errorf("%s: %q found after </head>", name, head)
		}
	}
	if bodyStart != "" {
		afterOpen := page[bodyOpen[1]:]
		idx := strings.Index(afterOpen, bodyStart)
		if idx < 0 {
			t.Errorf("%s: %q not after <body>", name, bodyStart)
		} else {
			between := strings.TrimSpace(afterOpen[:idx])
			if between != "" {
				t.Errorf("%s: %q is not the first content in <body>; saw %q before it", name, bodyStart, between)
			}
		}
		if strings.Contains(page[:bodyOpen[0]], bodyStart) {
			t.Errorf("%s: %q found before <body>", name, bodyStart)
		}
	}
	if bodyEnd != "" {
		at := strings.LastIndex(page[:bodyClose], bodyEnd)
		if at < 0 {
			t.Errorf("%s: %q not in <body>", name, bodyEnd)
			return
		}
		if strings.Contains(page[bodyClose:], bodyEnd) {
			t.Errorf("%s: %q found after </body>", name, bodyEnd)
		}
		between := strings.TrimSpace(page[at+len(bodyEnd) : bodyClose])
		if between != "" {
			t.Errorf("%s: %q is not last in <body>; saw %q after it", name, bodyEnd, between)
		}
	}
}

func indexFold(s, substr string) int {
	return strings.Index(strings.ToLower(s), strings.ToLower(substr))
}

func lastIndexFold(s, substr string) int {
	return strings.LastIndex(strings.ToLower(s), strings.ToLower(substr))
}
