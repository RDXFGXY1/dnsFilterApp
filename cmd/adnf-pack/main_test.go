package main

import (
	"github.com/RDXFGXY1/dns-filter-app/internal/sideapps"
	"os"
	"path/filepath"
	"testing"
)

func TestFocusOSPackageBuilds(t *testing.T) {
	raw, e := pack(filepath.Join("..", "..", "side-apps", "focus-os", "app"))
	if e != nil {
		t.Fatal(e)
	}
	p, e := sideapps.Parse(raw)
	if e != nil || p.Manifest.ID != "focus-os" {
		t.Fatalf("package: %+v %v", p, e)
	}
	if _, e = p.Render(); e != nil {
		t.Fatal(e)
	}
}
func TestPackRejectsNativeExecutable(t *testing.T) {
	dir := t.TempDir()
	if e := os.WriteFile(filepath.Join(dir, "run.exe"), []byte("MZ"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := pack(dir); e == nil {
		t.Fatal("native code accepted")
	}
}
