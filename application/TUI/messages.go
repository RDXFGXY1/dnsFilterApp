package main

import "time"

type authenticatedMsg struct{}
type snapshotMsg Snapshot
type tickMsg time.Time
type dashboardOpenedMsg struct{}
type errMsg struct{ err error }
type serviceActionMsg struct {
	action string
	err    error
}
type terminalSizeMsg struct{ width, height int }

func (e errMsg) Error() string { return e.err.Error() }
