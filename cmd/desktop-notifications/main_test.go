package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestReadOnlyCompanionDoesNotCreateDatabase(t *testing.T) {
	root := t.TempDir()
	if e := os.MkdirAll(filepath.Join(root, "configs"), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(root, "configs", "config.yaml"), []byte("database:\n  path: ./data/does-not-exist.db\n"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := run(root); e == nil {
		t.Fatal("missing database accepted")
	}
	if _, e := os.Stat(filepath.Join(root, "data", "does-not-exist.db")); !os.IsNotExist(e) {
		t.Fatal("read-only companion created a database")
	}
}
func TestCompanionReadsPathWithSpaces(t *testing.T) {
	root := filepath.Join(t.TempDir(), "a folder #1")
	if e := os.MkdirAll(filepath.Join(root, "configs"), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(root, "configs", "config.yaml"), []byte("database:\n  path: './deadlines #1.db'\n"), 0600); e != nil {
		t.Fatal(e)
	}
	db, e := sql.Open("sqlite", filepath.Join(root, "deadlines #1.db"))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(`CREATE TABLE example(id INTEGER)`); e != nil {
		t.Fatal(e)
	}
	db.Close()
	if e = run(root); e != nil {
		t.Fatal(e)
	}
}
