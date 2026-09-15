package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLegalPages(t *testing.T) {
	srv := httptest.NewServer(newRouter(newApp()))
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
	srv := httptest.NewServer(newRouter(newApp()))
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
			} {
				if !strings.Contains(page, want) {
					t.Errorf("missing footer link %q", want)
				}
			}
		})
	}
}

func getHTML(t *testing.T, srv *httptest.Server, path string, wantStatus int) string {
	t.Helper()
	resp, err := http.Get(srv.URL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if resp.StatusCode != wantStatus {
		t.Fatalf("GET %s: status %d, want %d", path, resp.StatusCode, wantStatus)
	}
	return string(body)
}
