//go:build !windows

package vault

func RemovableRoots() []string       { return nil }
func validRemovableRoot(string) bool { return false }
