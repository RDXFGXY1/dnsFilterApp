//go:build !windows

package vault

import "errors"

func protectLocal([]byte) ([]byte, error) {
	return nil, errors.New("email recovery requires Windows machine protection")
}
func unprotectLocal([]byte) ([]byte, error) {
	return nil, errors.New("email recovery requires Windows machine protection")
}
