//go:build linux

package connectionfreeze

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLinuxDropPacketsReadsFreezeCounter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nft")
	script := `#!/bin/sh
printf '%s' '{"nftables":[{"rule":{"comment":"dns-filter-connection-freeze","expr":[{"counter":{"packets":7,"bytes":448}},{"drop":null}]}}]}'
`
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	old := nftBinary
	nftBinary = path
	t.Cleanup(func() { nftBinary = old })
	packets, err := linuxDropPackets()
	if err != nil || packets != 7 {
		t.Fatalf("packets=%d err=%v", packets, err)
	}
}

func TestLinuxBackendEmitsOnlyCounterDeltas(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nft")
	count := filepath.Join(dir, "count")
	script := `#!/bin/sh
n=7
if [ -f "` + count + `" ]; then n=10; fi
touch "` + count + `"
printf '{"nftables":[{"rule":{"comment":"dns-filter-connection-freeze","expr":[{"counter":{"packets":%s,"bytes":1}}]}}]}' "$n"
`
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	old := nftBinary
	nftBinary = path
	t.Cleanup(func() { nftBinary = old })
	b := &linuxBackend{}
	first, err := b.Events()
	if err != nil || len(first) != 1 || b.lastPackets != 7 {
		t.Fatalf("first=%+v packets=%d err=%v", first, b.lastPackets, err)
	}
	second, err := b.Events()
	if err != nil || len(second) != 1 || b.lastPackets != 10 {
		t.Fatalf("second=%+v packets=%d err=%v", second, b.lastPackets, err)
	}
}
