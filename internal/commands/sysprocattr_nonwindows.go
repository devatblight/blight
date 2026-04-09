//go:build !windows

package commands

import "syscall"

func hiddenSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}
