package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLLMSTxtRouteServesStockGuide(t *testing.T) {
	srv := httptest.NewServer(newRouter(newAppConfig(false)))
	t.Cleanup(srv.Close)

	status, contentType, text := getResponse(t, srv.URL+"/llms.txt")
	if status != http.StatusOK {
		t.Fatalf("status %d", status)
	}
	if contentType != "text/plain; charset=utf-8" {
		t.Fatalf("content type %q", contentType)
	}
	for _, want := range []string{
		"# Fibo Planner",
		"POST /rooms",
		"/ws/{roomID}",
		`{"name":"Ada"}`,
		`{"points":"8"}`,
		`"admin":"load-next-topic"`,
		"???",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("stock /llms.txt missing %q", want)
		}
	}
	if strings.Contains(text, "<html") {
		t.Fatal("llms.txt should be plain text")
	}
}

func TestLoadLLMSTxtUsesCustomFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	const custom = "# Custom guide\n\nUse the hosted planner at https://planner.example.com/.\n"
	writeSnippet(t, dir, customLLMSFile, custom)

	got, err := loadLLMSTxt(dir)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != custom {
		t.Fatalf("got %q", got)
	}
	if strings.Contains(string(got), "POST /rooms") {
		t.Fatal("custom llms.txt should replace the stock guide")
	}
}

func TestLoadLLMSTxtEmptyCustomFileReplacesStock(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeSnippet(t, dir, customLLMSFile, "")

	got, err := loadLLMSTxt(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("empty custom llms.txt should replace stock, got %q", got)
	}
}

func TestLoadLLMSTxtMissingCustomFileKeepsStock(t *testing.T) {
	t.Parallel()

	got, err := loadLLMSTxt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "POST /rooms") {
		t.Fatal("missing custom llms.txt should keep the stock guide")
	}
}

func TestLoadLLMSTxtUnsetDirKeepsStock(t *testing.T) {
	t.Parallel()

	for _, dir := range []string{"", "   "} {
		got, err := loadLLMSTxt(dir)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(got), "# Fibo Planner") {
			t.Fatalf("dir %q: expected stock guide", dir)
		}
	}
}

func TestMustLoadLLMSTxtReportsReadError(t *testing.T) {
	origLoad, origFatal := loadLLMS, fatalf
	t.Cleanup(func() {
		loadLLMS = origLoad
		fatalf = origFatal
	})
	loadLLMS = func(string) ([]byte, error) {
		return nil, errors.New("disk")
	}
	var message string
	fatalf = func(format string, args ...any) {
		message = fmt.Sprintf(format, args...)
	}

	if got := mustLoadLLMSTxt(); got != nil {
		t.Fatalf("got %q", got)
	}
	if message != "llms.txt: disk" {
		t.Fatalf("message %q", message)
	}
}

func TestLoadLLMSTxtNotADirectory(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := loadLLMSTxt(path)
	if err == nil {
		t.Fatal("expected error when custom HTML path is not a directory")
	}
}
