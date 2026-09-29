package main

import (
	_ "embed"
	"log"
	"net/http"
	"os"
	"strings"
)

const customLLMSFile = "llms.txt"

//go:embed llms.txt
var stockLLMSTxt []byte

var llmsBody = mustLoadLLMSTxt()

func mustLoadLLMSTxt() []byte {
	b, err := loadLLMSTxt(os.Getenv(customHTMLDirEnv))
	if err != nil {
		log.Fatalf("llms.txt: %v", err)
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
