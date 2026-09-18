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

	for _, name := range []string{"index.html", "room.html", "room_not_found.html", "disclaimer.html", "privacy.html", "terms.html"} {
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

func TestParseAppTemplatesCustomLegalBodyReplacesCopyOnly(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeSnippet(t, dir, customHeadFile, `<!--HEAD-MARK-->`)
	writeSnippet(t, dir, customBodyStartFile, `<span id="BODY-START-MARK"></span>`)
	writeSnippet(t, dir, customBodyEndFile, `<span id="BODY-END-MARK"></span>`)
	writeSnippet(t, dir, customDisclaimerFile, `<p id="CUSTOM-DISCLAIMER">Hosted disclaimer.</p>`)
	writeSnippet(t, dir, customPrivacyFile, `<p id="CUSTOM-PRIVACY">Hosted privacy.</p>`)
	writeSnippet(t, dir, customTermsFile, `<p id="CUSTOM-TERMS">Hosted terms.</p>`)

	tmpl, err := parseAppTemplates(tplFS, dir)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	cases := []struct {
		name    string
		custom  string
		stock   string
		heading string
		crumb   string
	}{
		{
			name:    "disclaimer.html",
			custom:  `id="CUSTOM-DISCLAIMER">Hosted disclaimer.`,
			stock:   "without warranties of any kind",
			heading: "<h1>Disclaimer</h1>",
			crumb:   `Fibo Planner</a> · Disclaimer`,
		},
		{
			name:    "privacy.html",
			custom:  `id="CUSTOM-PRIVACY">Hosted privacy.`,
			stock:   "does not use advertising trackers",
			heading: "<h1>Privacy Policy</h1>",
			crumb:   `Fibo Planner</a> · Privacy Policy`,
		},
		{
			name:    "terms.html",
			custom:  `id="CUSTOM-TERMS">Hosted terms.`,
			stock:   "use the service responsibly",
			heading: "<h1>Terms of Use</h1>",
			crumb:   `Fibo Planner</a> · Terms of Use`,
		},
	}
	for _, tc := range cases {
		page := executeNamed(t, tmpl, tc.name)
		assertSnippetPositions(t, tc.name, page,
			"<!--HEAD-MARK-->",
			`<span id="BODY-START-MARK"></span>`,
			`<span id="BODY-END-MARK"></span>`,
		)
		if !strings.Contains(page, tc.custom) {
			t.Errorf("%s: missing custom body copy", tc.name)
		}
		if strings.Contains(page, tc.stock) {
			t.Errorf("%s: stock body copy should have been replaced", tc.name)
		}
		if !strings.Contains(page, tc.heading) {
			t.Errorf("%s: heading should remain", tc.name)
		}
		if !strings.Contains(page, tc.crumb) {
			t.Errorf("%s: crumb should remain", tc.name)
		}
		if !strings.Contains(page, `href="/disclaimer">Disclaimer</a>`) {
			t.Errorf("%s: site footer should remain", tc.name)
		}
	}

	index := executeNamed(t, tmpl, "index.html")
	if strings.Contains(index, "Hosted disclaimer") || strings.Contains(index, "Hosted privacy") || strings.Contains(index, "Hosted terms") {
		t.Fatal("custom legal body copy leaked onto index.html")
	}
}

func TestParseAppTemplatesCustomLegalBodyOneFileLeavesOthersStock(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeSnippet(t, dir, customDisclaimerFile, `<p>Only disclaimer is custom.</p>`)

	tmpl, err := parseAppTemplates(tplFS, dir)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	disclaimer := executeNamed(t, tmpl, "disclaimer.html")
	if !strings.Contains(disclaimer, "Only disclaimer is custom.") {
		t.Fatal("disclaimer missing custom copy")
	}
	if strings.Contains(disclaimer, "without warranties of any kind") {
		t.Fatal("disclaimer still has stock copy")
	}

	privacy := executeNamed(t, tmpl, "privacy.html")
	if !strings.Contains(privacy, "does not use advertising trackers") {
		t.Fatal("privacy should keep stock copy")
	}
	terms := executeNamed(t, tmpl, "terms.html")
	if !strings.Contains(terms, "use the service responsibly") {
		t.Fatal("terms should keep stock copy")
	}
}

func TestParseAppTemplatesEmptyCustomLegalBodyClearsStockCopy(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeSnippet(t, dir, customDisclaimerFile, "")

	tmpl, err := parseAppTemplates(tplFS, dir)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	page := executeNamed(t, tmpl, "disclaimer.html")
	if strings.Contains(page, "without warranties of any kind") {
		t.Fatal("empty custom disclaimer.html should replace stock copy")
	}
	if !strings.Contains(page, "<h1>Disclaimer</h1>") {
		t.Fatal("heading should remain when custom body is empty")
	}
	if !strings.Contains(page, `href="/disclaimer">Disclaimer</a>`) {
		t.Fatal("site footer should remain when custom body is empty")
	}
}

func TestParseAppTemplatesCustomLegalBodyTemplateActionsAreLiteral(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeSnippet(t, dir, customDisclaimerFile, `{{.RoomID}}<script>window.__fiboLegal=1</script>`)

	tmpl, err := parseAppTemplates(tplFS, dir)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	page := executeNamed(t, tmpl, "disclaimer.html")
	if !strings.Contains(page, "{{.RoomID}}") {
		t.Fatalf("expected literal {{.RoomID}} in custom legal body, got:\n%s", page)
	}
	if !strings.Contains(page, "<script>window.__fiboLegal=1</script>") {
		t.Fatal("script in custom legal body was escaped or dropped")
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
		data = lobbyPageData{}
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
	case "disclaimer.html", "privacy.html", "terms.html":
		data = nil
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
	disclaimer := executeNamed(t, tmpl, "disclaimer.html")
	if !strings.Contains(disclaimer, "<title>Disclaimer · Fibo Planner</title>") {
		t.Fatal("disclaimer.html missing title")
	}
	if !strings.Contains(disclaimer, "without warranties of any kind") {
		t.Fatal("disclaimer.html missing stock copy")
	}
	privacy := executeNamed(t, tmpl, "privacy.html")
	if !strings.Contains(privacy, "<title>Privacy Policy · Fibo Planner</title>") {
		t.Fatal("privacy.html missing title")
	}
	if !strings.Contains(privacy, "does not use advertising trackers") {
		t.Fatal("privacy.html missing stock copy")
	}
	terms := executeNamed(t, tmpl, "terms.html")
	if !strings.Contains(terms, "<title>Terms of Use · Fibo Planner</title>") {
		t.Fatal("terms.html missing title")
	}
	if !strings.Contains(terms, "use the service responsibly") {
		t.Fatal("terms.html missing stock copy")
	}
	for _, page := range []string{index, room, missing, disclaimer, privacy, terms} {
		for _, want := range []string{
			`href="https://github.com/siakhooi/fibo-planner" target="_blank" rel="noopener noreferrer">GitHub</a>`,
			`href="/disclaimer">Disclaimer</a>`,
			`href="/privacy">Privacy Policy</a>`,
			`href="/terms">Terms of Use</a>`,
		} {
			if !strings.Contains(page, want) {
				t.Errorf("page missing footer link %q", want)
			}
		}
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
