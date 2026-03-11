//go:build !windows

package tui

/*
setSysProcAttr starts Tilt in its own process group (Setpgid: true).
This isolates it so the terminal's raw-mode Ctrl+C key event is not
automatically delivered to the child as SIGINT.
 */
func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
