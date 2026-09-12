package main

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	customHTMLDirEnv    = "FIBO_PLANNER_CUSTOM_HTML_DIR"
	customHeadFile      = "head.html"
	customBodyStartFile = "body-start.html"
	customBodyEndFile   = "body-end.html"
)

//go:embed *.html
var tplFS embed.FS

var (
	reCloseHead = regexp.MustCompile(`(?i)</head>`)
	reOpenBody  = regexp.MustCompile(`(?i)<body\b[^>]*>`)
	reCloseBody = regexp.MustCompile(`(?i)</body>`)
)

type customHTML struct {
	Head      string
	BodyStart string
	BodyEnd   string
}

var tmpl = mustParseAppTemplates()

func mustParseAppTemplates() *template.Template {
	t, err := parseAppTemplates(tplFS, os.Getenv(customHTMLDirEnv))
	if err != nil {
		log.Fatalf("html templates: %v", err)
	}
	return t
}

func parseAppTemplates(htmlFS fs.FS, customDir string) (*template.Template, error) {
	c, err := loadCustomHTML(customDir)
	if err != nil {
		return nil, err
	}
	root := template.New("").Funcs(template.FuncMap{
		"customHead":      func() template.HTML { return template.HTML(c.Head) },
		"customBodyStart": func() template.HTML { return template.HTML(c.BodyStart) },
		"customBodyEnd":   func() template.HTML { return template.HTML(c.BodyEnd) },
	})
	names, err := fs.Glob(htmlFS, "*.html")
	if err != nil {
		return nil, err
	}
	for _, name := range names {
		b, err := fs.ReadFile(htmlFS, name)
		if err != nil {
			return nil, err
		}
		src := applyCustomHTMLSlots(string(b))
		if _, err := root.New(name).Parse(src); err != nil {
			return nil, fmt.Errorf("parse %s: %w", name, err)
		}
	}
	return root, nil
}

func loadCustomHTML(dir string) (customHTML, error) {
	var c customHTML
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return c, nil
	}

	fi, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("%s=%s does not exist; skipping custom HTML", customHTMLDirEnv, dir)
			return c, nil
		}
		return c, fmt.Errorf("%s: %w", customHTMLDirEnv, err)
	}
	if !fi.IsDir() {
		return c, fmt.Errorf("%s=%s is not a directory", customHTMLDirEnv, dir)
	}

	log.Printf("loading custom HTML from %s", dir)
	c.Head, err = readCustomSnippet(dir, customHeadFile)
	if err != nil {
		return c, err
	}
	c.BodyStart, err = readCustomSnippet(dir, customBodyStartFile)
	if err != nil {
		return c, err
	}
	c.BodyEnd, err = readCustomSnippet(dir, customBodyEndFile)
	if err != nil {
		return c, err
	}
	return c, nil
}

func readCustomSnippet(dir, name string) (string, error) {
	path := filepath.Join(dir, name)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	log.Printf("loaded custom HTML %s", path)
	return string(b), nil
}

func applyCustomHTMLSlots(src string) string {
	src = insertBeforeLast(src, reCloseHead, "{{customHead}}\n")
	src = insertAfterFirst(src, reOpenBody, "\n{{customBodyStart}}")
	src = insertBeforeLast(src, reCloseBody, "{{customBodyEnd}}\n")
	return src
}

func insertBeforeLast(src string, re *regexp.Regexp, fragment string) string {
	locs := re.FindAllStringIndex(src, -1)
	if len(locs) == 0 {
		return src
	}
	i := locs[len(locs)-1][0]
	return src[:i] + fragment + src[i:]
}

func insertAfterFirst(src string, re *regexp.Regexp, fragment string) string {
	loc := re.FindStringIndex(src)
	if loc == nil {
		return src
	}
	i := loc[1]
	return src[:i] + fragment + src[i:]
}
