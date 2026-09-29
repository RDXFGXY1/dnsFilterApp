//go:build !windows

package downloadguard

func signatureInfo(string) (string, string) { return "unavailable", "" }
