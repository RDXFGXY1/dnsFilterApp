//go:build !windows

package localdomain

import "errors"

func Ready(string) bool  { return false }
func Apply(string) error { return errors.New("local domain setup is currently available on Windows") }
