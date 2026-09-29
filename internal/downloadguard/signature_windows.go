//go:build windows

package downloadguard

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
)

func signatureInfo(path string) (string, string) {
	script := `$s=Get-AuthenticodeSignature -LiteralPath $env:DG_FILE;$o=[pscustomobject]@{Status=$s.Status.ToString();Publisher=if($s.SignerCertificate){$s.SignerCertificate.Subject}else{''}};$o|ConvertTo-Json -Compress`
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.Env = append(os.Environ(), "DG_FILE="+path)
	out, err := cmd.Output()
	if err != nil {
		return "unavailable", ""
	}
	var v struct{ Status, Publisher string }
	if json.Unmarshal(out, &v) != nil {
		return "unavailable", ""
	}
	switch strings.ToLower(v.Status) {
	case "valid":
		return "valid", v.Publisher
	case "notsigned":
		return "unsigned", ""
	default:
		return "invalid", v.Publisher
	}
}
