package api

import "testing"

func TestProfileChanges(t *testing.T) {
	current := networkProfileState{Filtering: true, Apps: map[string]bool{"gaming": false}, Categories: map[string]bool{"adult": true}, Whitelist: []string{"safe.test"}}
	target := networkProfileState{Filtering: false, Apps: map[string]bool{"gaming": true}, Categories: map[string]bool{"adult": true}, Whitelist: []string{"*.spotify.com"}}
	changes := profileChanges(current, target)
	if changes["filtering_changes"] != true || changes["app_changes"] != 1 || changes["category_changes"] != 0 || changes["whitelist_added"] != 1 || changes["whitelist_removed"] != 1 {
		t.Fatalf("unexpected profile preview: %#v", changes)
	}
}
