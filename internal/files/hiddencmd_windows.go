//go:build windows

package files

import "syscall"

// HiddenCmd returns process attributes for launching helper commands without a visible console on Windows.
func HiddenCmd(name string, args ...string) *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: true}
}
