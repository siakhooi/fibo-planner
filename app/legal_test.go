package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/siakhooi/fibo-planner/app/versioninfo"
)

func TestLegalPages(t *testing.T) {
	srv := httptest.NewServer(newRouter(newAppConfig(false)))
	t.Cleanup(srv.Close)

	cases := []struct {
		path string
		want []string
	}{
		{
			path: "/disclaimer",
			want: []string{
				"<title>Disclaimer · Fibo Planner</title>",
				"without warranties of any kind",
				"MIT License",
			},
		},
		{
			path: "/privacy",
			want: []string{
				"<title>Privacy Policy · Fibo Planner</title>",
				"Last updated: September 2026",
				"does not use advertising trackers",
				"cdn.jsdelivr.net",
				"visitor-count badge",
				"process access log does not record",
				"Changes to This Policy",
			},
		},
		{
			path: "/terms",
			want: []string{
				"<title>Terms of Use · Fibo Planner</title>",
				"use the service responsibly",
				"do not replace the terms of the software license",
				"published on this page",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			page := getHTML(t, srv, tc.path, http.StatusOK)
			for _, want := range tc.want {
				if !strings.Contains(page, want) {
					t.Errorf("missing %q", want)
				}
			}
		})
	}
}

func TestPagesShareLegalFooter(t *testing.T) {
	srv := httptest.NewServer(newRouter(newAppConfig(false)))
	t.Cleanup(srv.Close)

	roomID := createRoom(t, srv, "sprint")
	paths := []struct {
		path string
		code int
	}{
		{path: "/", code: http.StatusOK},
		{path: "/" + roomID, code: http.StatusOK},
		{path: "/000000", code: http.StatusNotFound},
		{path: "/disclaimer", code: http.StatusOK},
		{path: "/privacy", code: http.StatusOK},
		{path: "/terms", code: http.StatusOK},
	}
	for _, tc := range paths {
		t.Run(tc.path, func(t *testing.T) {
			page := getHTML(t, srv, tc.path, tc.code)
			for _, want := range []string{
				`href="/disclaimer">Disclaimer</a>`,
				`href="/privacy">Privacy Policy</a>`,
				`href="/terms">Terms of Use</a>`,
				`class="app-version">` + versioninfo.Version,
			} {
				if !strings.Contains(page, want) {
					t.Errorf("missing footer link %q", want)
				}
			}
		})
	}
}

func TestLegalPageViewFor(t *testing.T) {
	cases := []struct {
		name string
		want legalPageView
	}{
		{
			name: customDisclaimerFile,
			want: legalPageView{File: customDisclaimerFile, Title: "Disclaimer · Fibo Planner", Crumb: "Disclaimer", Heading: "Disclaimer"},
		},
		{
			name: customPrivacyFile,
			want: legalPageView{File: customPrivacyFile, Title: "Privacy Policy · Fibo Planner", Crumb: "Privacy Policy", Heading: "Privacy Policy"},
		},
		{
			name: customTermsFile,
			want: legalPageView{File: customTermsFile, Title: "Terms of Use · Fibo Planner", Crumb: "Terms of Use", Heading: "Terms of Use"},
		},
	}
	for _, tc := range cases {
		got, ok := legalPageViewFor(tc.name)
		if !ok || got != tc.want {
			t.Fatalf("legalPageViewFor(%q) = %+v, %v", tc.name, got, ok)
		}
	}

	if got, ok := legalPageViewFor("index.html"); ok || got != (legalPageView{}) {
		t.Fatalf("unknown page = %+v, %v", got, ok)
	}
}

func TestLegalPageUnknownNamePanics(t *testing.T) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected panic")
		}
		if recovered != "unknown legal page: index.html" {
			t.Fatalf("panic %v", recovered)
		}
	}()
	legalPage("index.html")
}
