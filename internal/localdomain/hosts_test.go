package localdomain

import (
	"strings"
	"testing"
)

func TestRewriteHostsChangesOnlyManagedDashboardEntry(t *testing.T) {
	original := []byte("127.0.0.1 localhost\r\n127.0.0.1 guard.nullstudio.org # DNS Filter local dashboard\r\n203.0.113.5 unrelated.example\r\n")
	updated, err := rewriteHosts(original, "myguard.example.org")
	if err != nil {
		t.Fatal(err)
	}
	text := string(updated)
	if strings.Contains(text, "guard.nullstudio.org") || !strings.Contains(text, "127.0.0.1 myguard.example.org "+marker) || !strings.Contains(text, "203.0.113.5 unrelated.example") {
		t.Fatalf("wrong hosts change: %q", text)
	}
	if !containsLocalEntry(updated, "myguard.example.org") {
		t.Fatal("new local entry missing")
	}
	if _, err := rewriteHosts([]byte("198.51.100.7 myguard.example.org\r\n"), "myguard.example.org"); err == nil {
		t.Fatal("conflicting entry accepted")
	}
}
