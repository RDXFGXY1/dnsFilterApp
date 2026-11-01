// Package sideapps installs independent, sandboxed HTML applications from ADNF archives.
package sideapps

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"
)

const MaxPackageBytes = 5 << 20
const MaxExpandedBytes = 16 << 20
const MaxStorageBytes = 1 << 20

var idPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{1,63}$`)
var versionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:-[a-zA-Z0-9.-]+)?$`)
var ErrConflict = errors.New("this app is already installed; explicitly approve replacement")
var ErrForbidden = errors.New("app is disabled or does not have the required permission")

type Manifest struct {
	Format      int      `json:"format"`
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Author      string   `json:"author"`
	Description string   `json:"description"`
	Entry       string   `json:"entry"`
	Permissions []string `json:"permissions"`
}
type Package struct {
	Manifest      Manifest          `json:"manifest"`
	SHA256        string            `json:"sha256"`
	FileCount     int               `json:"file_count"`
	ExpandedBytes int               `json:"expanded_bytes"`
	Files         map[string][]byte `json:"-"`
	Archive       []byte            `json:"-"`
}

func safePath(name string) bool {
	if name == "" || len(name) > 240 || strings.ContainsAny(name, "\\:\x00") || strings.HasPrefix(name, "/") || path.Clean(name) != name {
		return false
	}
	for _, part := range strings.Split(name, "/") {
		if strings.HasPrefix(part, ".") || strings.TrimSpace(part) != part || strings.HasSuffix(part, ".") {
			return false
		}
	}
	return true
}
func Parse(raw []byte) (*Package, error) {
	if len(raw) == 0 || len(raw) > MaxPackageBytes {
		return nil, errors.New("ADNF package must be between 1 byte and 5 MB")
	}
	z, e := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if e != nil {
		return nil, errors.New("invalid ADNF archive")
	}
	if len(z.File) == 0 || len(z.File) > 128 {
		return nil, errors.New("an ADNF package may contain at most 128 files")
	}
	p := &Package{Files: map[string][]byte{}, Archive: raw}
	sum := sha256.Sum256(raw)
	p.SHA256 = hex.EncodeToString(sum[:])
	seen := map[string]bool{}
	for _, f := range z.File {
		name := f.Name
		if f.FileInfo().IsDir() {
			name = strings.TrimSuffix(name, "/")
			if !safePath(name) {
				return nil, fmt.Errorf("unsafe directory: %q", name)
			}
			continue
		}
		if !safePath(name) || !f.Mode().IsRegular() {
			return nil, fmt.Errorf("unsafe file: %q", name)
		}
		key := strings.ToLower(name)
		if seen[key] {
			return nil, errors.New("duplicate or case-ambiguous package path")
		}
		seen[key] = true
		switch strings.ToLower(path.Ext(name)) {
		case ".html", ".css", ".js", ".json", ".txt", ".png", ".jpg", ".jpeg", ".gif", ".webp", ".ico":
		default:
			return nil, fmt.Errorf("unsupported package file: %s", name)
		}
		if f.UncompressedSize64 > 4<<20 {
			return nil, errors.New("individual files must be smaller than 4 MB")
		}
		r, e := f.Open()
		if e != nil {
			return nil, e
		}
		b, e := io.ReadAll(io.LimitReader(r, 4<<20+1))
		r.Close()
		if e != nil {
			return nil, e
		}
		if len(b) > 4<<20 {
			return nil, errors.New("expanded file is too large")
		}
		p.ExpandedBytes += len(b)
		if p.ExpandedBytes > MaxExpandedBytes {
			return nil, errors.New("expanded package is larger than 16 MB")
		}
		p.Files[name] = b
	}
	m, ok := p.Files["direct.json"]
	if !ok || len(m) > 16<<10 {
		return nil, errors.New("package needs direct.json at its root (maximum 16 KB)")
	}
	decoder := json.NewDecoder(bytes.NewReader(m))
	decoder.DisallowUnknownFields()
	if e = decoder.Decode(&p.Manifest); e != nil {
		return nil, fmt.Errorf("invalid direct.json: %w", e)
	}
	if decoder.Decode(new(any)) != io.EOF {
		return nil, errors.New("direct.json must contain exactly one JSON object")
	}
	a := &p.Manifest
	a.Name = strings.TrimSpace(a.Name)
	if a.Format != 1 || !idPattern.MatchString(a.ID) || !versionPattern.MatchString(a.Version) || a.Name == "" || len(a.Name) > 80 || len(a.Author) > 120 || len(a.Description) > 600 {
		return nil, errors.New("invalid manifest identity, format or version")
	}
	if !safePath(a.Entry) || path.Ext(a.Entry) != ".html" {
		return nil, errors.New("entry must be a packaged .html file")
	}
	if _, ok = p.Files[a.Entry]; !ok {
		return nil, errors.New("entry HTML file is missing")
	}
	if len(a.Permissions) > 1 {
		return nil, errors.New("duplicate or unsupported permission")
	}
	for _, permission := range a.Permissions {
		if permission != "storage" {
			return nil, fmt.Errorf("unsupported permission: %s", permission)
		}
	}
	if a.Permissions == nil {
		a.Permissions = []string{}
	}
	p.FileCount = len(p.Files)
	return p, nil
}
