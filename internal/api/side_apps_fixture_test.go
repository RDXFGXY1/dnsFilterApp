package api

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func sideAppExampleArchive(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for _, name := range []string{"direct.json", "index.html", "app.js", "styles.css"} {
		raw, e := os.ReadFile(filepath.Join("..", "..", "side-apps", "focus-os", "app", name))
		if e != nil {
			t.Fatal(e)
		}
		f, e := z.Create(name)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = f.Write(raw); e != nil {
			t.Fatal(e)
		}
	}
	if e := z.Close(); e != nil {
		t.Fatal(e)
	}
	return b.Bytes()
}
