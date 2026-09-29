// Package version exposes the single product version shared by the service,
// dashboard API, desktop application, and release packaging.
package version

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var raw string

// Current is the canonical user-facing product version.
var Current = strings.TrimSpace(raw)
