//go:build windows

package tui

import (
	"os/exec"
	"syscall"
)

/*
setSysProcAttr starts Tilt in a new Windows process group
(CREATE_NEW_PROCESS_GROUP). This is required so that the console Ctrl+C
event arriving at the parent TUI is not automatically forwarded to the
child Tilt process. We manage the lifecycle explicitly via tilt down.
*/
func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}
