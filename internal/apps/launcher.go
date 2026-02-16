package apps

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

func Launch(app AppEntry) error {
	target := app.Path

	if app.IsLnk {
		target = app.LnkPath
	}

	var cmd *exec.Cmd

	if strings.HasSuffix(strings.ToLower(target), ".lnk") {
		cmd = exec.Command("cmd", "/c", "start", "", target)
	} else {
		cmd = exec.Command(target)
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x00000008, // DETACHED_PROCESS
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to launch %s: %w", app.Name, err)
	}

	return nil
}
