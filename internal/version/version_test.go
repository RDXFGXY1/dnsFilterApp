package version

import "testing"

func TestCurrentIsReleaseSafe(t *testing.T) {
	if Current != "0.1.0" {
		t.Fatalf("unexpected product version %q", Current)
	}
}
