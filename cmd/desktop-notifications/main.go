// desktop-notifications is a read-only local companion to the Windows tray app.
package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/RDXFGXY1/dns-filter-app/internal/config"
	"github.com/RDXFGXY1/dns-filter-app/internal/deadlines"
	_ "modernc.org/sqlite"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

func run(root string) error {
	if root == "" {
		return fmt.Errorf("provide the DNS Filter installation folder with -root")
	}
	cfg, e := config.Load(filepath.Join(root, "configs", "config.yaml"))
	if e != nil {
		return e
	}
	dbPath := cfg.Database.Path
	if !filepath.IsAbs(dbPath) {
		dbPath = filepath.Join(root, dbPath)
	}
	if _, e = os.Stat(dbPath); e != nil {
		return fmt.Errorf("the local database is not available: %w", e)
	}
	// URI-escape special characters while preserving the Windows drive prefix.
	query := url.Values{}
	query.Set("mode", "ro")
	query.Set("_pragma", "busy_timeout(2000)")
	dsn := "file:" + (&url.URL{Path: filepath.ToSlash(dbPath)}).EscapedPath() + "?" + query.Encode()
	db, e := sql.Open("sqlite", dsn)
	if e != nil {
		return e
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	items, e := deadlines.Collect(db, time.Now())
	if e != nil {
		return e
	}
	return json.NewEncoder(os.Stdout).Encode(items)
}
func main() {
	root := flag.String("root", "", "DNS Filter installation folder")
	flag.Parse()
	if e := run(*root); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
