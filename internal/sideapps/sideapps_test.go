package sideapps

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	_ "modernc.org/sqlite"
	"strings"
	"testing"
)

func testPackage(t *testing.T, id string, permission bool) []byte {
	t.Helper()
	permissions := []string{}
	if permission {
		permissions = append(permissions, "storage")
	}
	m := Manifest{Format: 1, ID: id, Name: "Test app", Version: "0.1.0", Entry: "index.html", Permissions: permissions}
	b, _ := json.Marshal(m)
	return archiveFiles(t, map[string][]byte{"direct.json": b, "index.html": []byte(`<!doctype html><html><head><link rel="stylesheet" href="app.css"></head><body><h1>Independent</h1><script src="app.js"></script></body></html>`), "app.css": []byte(`body{color:white}`), "app.js": []byte(`window.appStarted=true;`)})
}
func archiveFiles(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for n, raw := range files {
		w, e := z.Create(n)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = w.Write(raw); e != nil {
			t.Fatal(e)
		}
	}
	if e := z.Close(); e != nil {
		t.Fatal(e)
	}
	return b.Bytes()
}
func newTestManager(t *testing.T) *Manager {
	t.Helper()
	db, e := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close() })
	m, e := New(db)
	if e != nil {
		t.Fatal(e)
	}
	return m
}
func TestInstallUpdatePermissionsAndAppIsolation(t *testing.T) {
	m := newTestManager(t)
	a := testPackage(t, "first-app", true)
	p, e := Parse(a)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = m.Install(a, "wrong", false); e == nil {
		t.Fatal("unreviewed package installed")
	}
	if _, e = m.Install(a, p.SHA256, false); e != nil {
		t.Fatal(e)
	}
	if _, e = m.Install(a, p.SHA256, false); !errors.Is(e, ErrConflict) {
		t.Fatalf("duplicate: %v", e)
	}
	if e = m.WriteStorage("first-app", json.RawMessage(`{"notes":"private"}`)); e != nil {
		t.Fatal(e)
	}
	b := testPackage(t, "second-app", true)
	p2, _ := Parse(b)
	if _, e = m.Install(b, p2.SHA256, false); e != nil {
		t.Fatal(e)
	}
	raw, e := m.ReadStorage("second-app")
	if e != nil || string(raw) != "null" {
		t.Fatalf("cross-app leak: %s %v", raw, e)
	}
	if _, e = m.Install(a, p.SHA256, true); e != nil {
		t.Fatal(e)
	}
	raw, e = m.ReadStorage("first-app")
	if e != nil || !strings.Contains(string(raw), "private") {
		t.Fatal("update lost app data")
	}
	if e = m.SetEnabled("first-app", false); e != nil {
		t.Fatal(e)
	}
	if _, e = m.Package("first-app"); !errors.Is(e, ErrForbidden) {
		t.Fatalf("disabled app launches: %v", e)
	}
	if e = m.WriteStorage("first-app", json.RawMessage(`{}`)); !errors.Is(e, ErrForbidden) {
		t.Fatalf("disabled app writes: %v", e)
	}
	if e = m.SetEnabled("first-app", true); e != nil {
		t.Fatal(e)
	}
	if e = m.Remove("first-app"); e != nil {
		t.Fatal(e)
	}
	if _, e = m.ReadStorage("first-app"); !errors.Is(e, sql.ErrNoRows) {
		t.Fatalf("uninstalled storage: %v", e)
	}
	// Reinstall starts clean; other apps' state is preserved.
	if _, e = m.Install(a, p.SHA256, false); e != nil {
		t.Fatal(e)
	}
	raw, e = m.ReadStorage("first-app")
	if e != nil || string(raw) != "null" {
		t.Fatalf("data survived explicit uninstall: %s %v", raw, e)
	}
	apps, e := m.List()
	if e != nil || len(apps) != 2 {
		t.Fatalf("apps: %+v %v", apps, e)
	}
}
func TestUndeclaredStorageAndQuota(t *testing.T) {
	m := newTestManager(t)
	raw := testPackage(t, "no-storage", false)
	p, _ := Parse(raw)
	if _, e := m.Install(raw, p.SHA256, false); e != nil {
		t.Fatal(e)
	}
	if _, e := m.ReadStorage("no-storage"); !errors.Is(e, ErrForbidden) {
		t.Fatal("undeclared permission accepted")
	}
	if e := m.WriteStorage("no-storage", json.RawMessage(`{}`)); !errors.Is(e, ErrForbidden) {
		t.Fatal("undeclared write accepted")
	}
	raw = testPackage(t, "with-storage", true)
	p, _ = Parse(raw)
	if _, e := m.Install(raw, p.SHA256, false); e != nil {
		t.Fatal(e)
	}
	for _, value := range []json.RawMessage{json.RawMessage(`{invalid`), json.RawMessage(`"` + strings.Repeat("x", MaxStorageBytes) + `"`)} {
		if e := m.WriteStorage("with-storage", value); e == nil {
			t.Fatal("invalid storage accepted")
		}
	}
}
func TestUnsafeArchivePathsAndFileTypes(t *testing.T) {
	for _, name := range []string{"../index.html", "/index.html", `C:\evil.exe`, `folder\evil.js`, "folder/../../evil.js", ".hidden.js", "app.exe", "app.dll", "app.svg", "folder/./app.js", "folder /app.js"} {
		raw := archiveFiles(t, map[string][]byte{name: []byte("x")})
		if _, e := Parse(raw); e == nil {
			t.Errorf("accepted unsafe file %q", name)
		}
	}
	raw := archiveFiles(t, map[string][]byte{"direct.json": []byte(`{"format":1,"id":"my-app","name":"A","version":"1.0.0","entry":"index.html","permissions":["dns.write"]}`), "index.html": []byte("hello")})
	if _, e := Parse(raw); e == nil {
		t.Fatal("unknown permission accepted")
	}
	raw = archiveFiles(t, map[string][]byte{"huge.txt": bytes.Repeat([]byte("a"), 4<<20+1)})
	if _, e := Parse(raw); e == nil {
		t.Fatal("oversized expanded file accepted")
	}
	// Case aliases are ambiguous on Windows and must be rejected on every OS.
	raw = archiveFiles(t, map[string][]byte{"app.js": []byte("x"), "APP.js": []byte("y")})
	if _, e := Parse(raw); e == nil {
		t.Fatal("case aliases accepted")
	}
}
func TestRendererBundlesResourcesAndInjectsBridge(t *testing.T) {
	p, e := Parse(testPackage(t, "render-app", true))
	if e != nil {
		t.Fatal(e)
	}
	out, e := p.Render()
	if e != nil {
		t.Fatal(e)
	}
	s := string(out)
	for _, fragment := range []string{"adnf:ready", "window.appStarted=true", "body{color:white}"} {
		if !strings.Contains(s, fragment) {
			t.Errorf("missing %s", fragment)
		}
	}
	if strings.Contains(s, `src="app.js"`) || strings.Contains(s, `href="app.css"`) {
		t.Fatal("unbundled resources")
	}
	if strings.Contains(AppCSP, "allow-same-origin") || !strings.Contains(AppCSP, "connect-src 'none'") || !strings.Contains(AppCSP, "sandbox allow-scripts") {
		t.Fatal("unsafe runner policy")
	}
	p.Files[p.Manifest.Entry] = []byte(`<script src="https://example.org/app.js"></script>`)
	if _, e = p.Render(); e == nil {
		t.Fatal("external script accepted")
	}
}
