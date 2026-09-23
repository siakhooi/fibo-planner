package main

import "net/http"

func legalPage(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeHTML(w, http.StatusOK, name, nil)
	}
}
