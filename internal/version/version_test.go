package version

import "testing"

func TestCurrentIsReleaseSafe(t *testing.T) {
	if Current != "2.6.0-beta.1" {
		t.Fatalf("unexpected product version %q", Current)
	}
}
