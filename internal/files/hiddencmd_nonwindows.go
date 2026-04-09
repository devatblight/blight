//go:build !windows

package files

import "syscall"

// HiddenCmd returns portable process attributes for non-Windows targets.
func HiddenCmd(name string, args ...string) *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}
