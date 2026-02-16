package commands

import (
	"os/exec"
	"syscall"
)

type SystemCommand struct {
	ID       string
	Name     string
	Subtitle string
	Icon     string
	Keywords []string
}

var SystemCommands = []SystemCommand{
	{
		ID: "lock-screen", Name: "Lock Screen",
		Subtitle: "Lock this computer",
		Icon:     "🔒", Keywords: []string{"lock", "screen", "secure"},
	},
	{
		ID: "sleep", Name: "Sleep",
		Subtitle: "Put computer to sleep",
		Icon:     "💤", Keywords: []string{"sleep", "suspend", "standby"},
	},
	{
		ID: "shutdown", Name: "Shut Down",
		Subtitle: "Shut down this computer",
		Icon:     "⏻", Keywords: []string{"shutdown", "shut down", "power off", "turn off"},
	},
	{
		ID: "restart", Name: "Restart",
		Subtitle: "Restart this computer",
		Icon:     "🔄", Keywords: []string{"restart", "reboot"},
	},
	{
		ID: "recycle-bin", Name: "Empty Recycle Bin",
		Subtitle: "Permanently delete recycled files",
		Icon:     "🗑️", Keywords: []string{"recycle", "bin", "trash", "empty", "delete"},
	},
	{
		ID: "logout", Name: "Log Out",
		Subtitle: "Sign out of this account",
		Icon:     "🚪", Keywords: []string{"logout", "log out", "sign out", "signout"},
	},
}

func ExecuteSystemCommand(id string) error {
	switch id {
	case "lock-screen":
		return runHidden("rundll32.exe", "user32.dll,LockWorkStation")
	case "sleep":
		return runHidden("rundll32.exe", "powrprof.dll,SetSuspendState", "0", "1", "0")
	case "shutdown":
		return runHidden("shutdown.exe", "/s", "/t", "0")
	case "restart":
		return runHidden("shutdown.exe", "/r", "/t", "0")
	case "recycle-bin":
		return runHidden("cmd.exe", "/c", "rd", "/s", "/q", "C:\\$Recycle.Bin")
	case "logout":
		return runHidden("shutdown.exe", "/l")
	}
	return nil
}

func runHidden(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run()
}
