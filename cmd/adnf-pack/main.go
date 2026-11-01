// adnf-pack builds an independent HTML app into an installable ADNF archive.
package main

import (
	"archive/zip"
	"bytes"
	"flag"
	"fmt"
	"github.com/RDXFGXY1/dns-filter-app/internal/sideapps"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func pack(source string) ([]byte, error) {
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	err := filepath.WalkDir(source, func(file string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlinks are not supported: %s", file)
		}
		if entry.IsDir() {
			return nil
		}
		relative, e := filepath.Rel(source, file)
		if e != nil {
			return e
		}
		relative = filepath.ToSlash(relative)
		info, e := entry.Info()
		if e != nil {
			return e
		}
		if !info.Mode().IsRegular() || info.Size() > 4<<20 {
			return fmt.Errorf("invalid or oversized file: %s", relative)
		}
		raw, e := os.ReadFile(file)
		if e != nil {
			return e
		}
		header := &zip.FileHeader{Name: relative, Method: zip.Deflate}
		header.SetModTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
		header.SetMode(0600)
		writer, e := archive.CreateHeader(header)
		if e != nil {
			return e
		}
		_, e = writer.Write(raw)
		return e
	})
	if err != nil {
		return nil, err
	}
	if err = archive.Close(); err != nil {
		return nil, err
	}
	raw := buffer.Bytes()
	p, err := sideapps.Parse(raw)
	if err != nil {
		return nil, err
	}
	if _, err = p.Render(); err != nil {
		return nil, err
	}
	return raw, nil
}
func main() {
	source := flag.String("source", "", "folder containing direct.json and app files")
	output := flag.String("output", "", "destination .adnf package")
	flag.Parse()
	if *source == "" || !strings.EqualFold(filepath.Ext(*output), ".adnf") {
		fmt.Fprintln(os.Stderr, "Usage: adnf-pack -source path/to/app -output app.adnf")
		os.Exit(2)
	}
	raw, err := pack(*source)
	if err == nil {
		err = os.MkdirAll(filepath.Dir(*output), 0700)
	}
	if err == nil {
		err = os.WriteFile(*output, raw, 0600)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	p, _ := sideapps.Parse(raw)
	fmt.Printf("Built %s (%d bytes)\nApp: %s v%s\nSHA-256: %s\n", *output, len(raw), p.Manifest.Name, p.Manifest.Version, p.SHA256)
}
