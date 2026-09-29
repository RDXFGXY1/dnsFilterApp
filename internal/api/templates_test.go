package api

import (
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestWebTemplatesParseTogether(t *testing.T) {
	if _, err := template.ParseGlob("../../web/templates/*"); err != nil {
		t.Fatalf("web templates do not parse: %v", err)
	}
}

func TestEveryInlineScriptHasCSPNonce(t *testing.T) {
	files, err := filepath.Glob("../../web/templates/*")
	if err != nil {
		t.Fatal(err)
	}
	scriptTag := regexp.MustCompile(`(?i)<script(?:\s[^>]*)?>`)
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, tag := range scriptTag.FindAllString(string(data), -1) {
			if strings.Contains(strings.ToLower(tag), " src=") {
				continue
			}
			if !strings.Contains(tag, `nonce="{{ .csp_nonce }}"`) {
				t.Errorf("inline script without CSP nonce in %s: %s", filepath.Base(path), tag)
			}
		}
	}
}
