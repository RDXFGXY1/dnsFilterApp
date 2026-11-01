package autostart

import "testing"

func TestStartupModes(t *testing.T) {
	for _, mode := range []string{ModeBoot, ModeLogin, ModeManual} {
		if e := (Options{Mode: mode}).Validate(); e != nil {
			t.Fatal(e)
		}
	}
	for _, mode := range []string{"", "unknown", "login;whoami"} {
		if e := (Options{Mode: mode}).Validate(); e == nil {
			t.Fatalf("accepted %q", mode)
		}
	}
}
