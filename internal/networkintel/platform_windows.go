//go:build windows

package networkintel

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type windowsPlatform struct{}

func newPlatform() platform             { return windowsPlatform{} }
func (windowsPlatform) Supported() bool { return true }
func (windowsPlatform) Connections() ([]Connection, error) {
	script := `Get-NetTCPConnection -State Established -ErrorAction SilentlyContinue | ForEach-Object {$p=Get-Process -Id $_.OwningProcess -ErrorAction SilentlyContinue;[pscustomobject]@{Path=$p.Path;RemoteIP=$_.RemoteAddress;RemotePort=$_.RemotePort}} | ConvertTo-Json -Compress`
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("connection inventory failed: %s", strings.TrimSpace(string(out)))
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" || raw == "null" {
		return []Connection{}, nil
	}
	var list []struct {
		Path, RemoteIP string
		RemotePort     int
	}
	if raw[0] == '{' {
		var one struct {
			Path, RemoteIP string
			RemotePort     int
		}
		if err := json.Unmarshal(out, &one); err != nil {
			return nil, err
		}
		list = append(list, one)
	} else if err := json.Unmarshal(out, &list); err != nil {
		return nil, err
	}
	result := make([]Connection, 0, len(list))
	for _, x := range list {
		result = append(result, Connection{Path: x.Path, RemoteIP: x.RemoteIP, RemotePort: x.RemotePort, Protocol: "TCP"})
	}
	return result, nil
}
