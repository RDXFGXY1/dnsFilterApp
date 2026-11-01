package autostart

import "fmt"

const (
	ModeBoot   = "boot"
	ModeLogin  = "login"
	ModeManual = "manual"
)

type Options struct {
	Mode       string `json:"mode"`
	RecoverDNS bool   `json:"recover_dns"`
}
type Status struct {
	ReadOK            bool   `json:"read_ok"`
	Supported         bool   `json:"supported"`
	Enabled           bool   `json:"enabled"`
	Mode              string `json:"mode"`
	RecoverDNS        bool   `json:"recover_dns"`
	RecoverySupported bool   `json:"recovery_supported"`
	LoginSupported    bool   `json:"login_supported"`
	Executable        string `json:"executable,omitempty"`
	WorkingDirectory  string `json:"working_directory,omitempty"`
	LastResult        uint32 `json:"last_result"`
	Problem           string `json:"problem,omitempty"`
}

func (o Options) Validate() error {
	switch o.Mode {
	case ModeBoot, ModeLogin, ModeManual:
		return nil
	default:
		return fmt.Errorf("startup mode must be boot, login, or manual")
	}
}
