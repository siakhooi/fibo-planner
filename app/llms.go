package main

import (
	_ "embed" // required so //go:embed can compile llms.txt into stockLLMSTxt; no embed symbol is referenced
	"net/http"
	"strings"
)

const customLLMSFile = "llms.txt"

//go:embed llms.txt
var stockLLMSTxt []byte

// loadLLMS reads the guide at startup. Tests replace it to exercise a read failure.
// llmsBody starts as the embedded guide. loadCustomContent replaces it.
var (
	loadLLMS = loadLLMSTxt
	llmsBody = mustLoadLLMSTxt()
)

func mustLoadLLMSTxt() []byte {
	b, err := loadLLMS("")
	if err != nil {
		fatalf("llms.txt: %v", err)
	}
	return b
}

// loadLLMSTxt returns the custom directory's llms.txt when that file exists.
// A missing file, or an unset directory, keeps the embedded guide. An empty
// custom file is used as-is.
func loadLLMSTxt(customDir string) ([]byte, error) {
	customDir = strings.TrimSpace(customDir)
	if customDir == "" {
		return stockLLMSTxt, nil
	}
	content, found, err := tryReadCustomSnippet(customDir, customLLMSFile)
	if err != nil {
		return nil, err
	}
	if !found {
		return stockLLMSTxt, nil
	}
	return []byte(content), nil
}

func serveLLMSTxt(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write(llmsBody)
}
