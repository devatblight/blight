//go:build darwin

package updater

import (
	"strings"
	"syscall"
)

func isInstallerAsset(name string) bool {
	return strings.HasSuffix(name, ".dmg") ||
		strings.HasSuffix(name, ".pkg")
}

func installerTempName() string {
	return "blight-installer"
}

func installerSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
