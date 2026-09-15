package main

import (
	"bytes"
	"net/http"
)

func legalPage(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		var buf bytes.Buffer
		if err := tmpl.ExecuteTemplate(&buf, name, nil); err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(buf.Bytes())
	}
}
