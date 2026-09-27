package main

import "net/http"

type legalPageView struct {
	File    string
	Title   string
	Crumb   string
	Heading string
}

func legalPageViewFor(name string) (legalPageView, bool) {
	switch name {
	case customDisclaimerFile:
		return legalPageView{
			File:    name,
			Title:   "Disclaimer · Fibo Planner",
			Crumb:   "Disclaimer",
			Heading: "Disclaimer",
		}, true
	case customPrivacyFile:
		return legalPageView{
			File:    name,
			Title:   "Privacy Policy · Fibo Planner",
			Crumb:   "Privacy Policy",
			Heading: "Privacy Policy",
		}, true
	case customTermsFile:
		return legalPageView{
			File:    name,
			Title:   "Terms of Use · Fibo Planner",
			Crumb:   "Terms of Use",
			Heading: "Terms of Use",
		}, true
	default:
		return legalPageView{}, false
	}
}

func legalPage(name string) http.HandlerFunc {
	view, ok := legalPageViewFor(name)
	if !ok {
		panic("unknown legal page: " + name)
	}
	return func(w http.ResponseWriter, _ *http.Request) {
		writeHTML(w, http.StatusOK, name, view)
	}
}
