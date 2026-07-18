package main

import (
	"embed"
	"html/template"
)

//go:embed *.html
var tplFS embed.FS

var tmpl = template.Must(template.ParseFS(tplFS, "*.html"))
