package commands

import (
	"os/exec"
)

type SystemCommand struct {
	ID       string
	Name     string
	Subtitle string
	Icon     string
	Keywords []string
}

var SystemCommands = []SystemCommand{
	// --- Power ---
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
		ID: "logout", Name: "Log Out",
		Subtitle: "Sign out of this account",
		Icon:     "🚪", Keywords: []string{"logout", "log out", "sign out", "signout"},
	},
	{
		ID: "hibernate", Name: "Hibernate",
		Subtitle: "Save session and power off",
		Icon:     "💾", Keywords: []string{"hibernate", "power"},
	},

	// --- Common System Apps ---
	{
		ID: "task-manager", Name: "Task Manager",
		Subtitle: "View running processes and performance",
		Icon:     "📊", Keywords: []string{"task", "manager", "processes", "cpu", "memory", "performance"},
	},
	{
		ID: "file-explorer", Name: "File Explorer",
		Subtitle: "Open Windows File Explorer",
		Icon:     "📁", Keywords: []string{"explorer", "files", "folder", "file manager"},
	},
	{
		ID: "calculator", Name: "Calculator",
		Subtitle: "Open Windows Calculator",
		Icon:     "🔢", Keywords: []string{"calculator", "calc", "math"},
	},
	{
		ID: "notepad", Name: "Notepad",
		Subtitle: "Open Notepad",
		Icon:     "📝", Keywords: []string{"notepad", "text", "editor"},
	},
	{
		ID: "snipping-tool", Name: "Snipping Tool",
		Subtitle: "Take a screenshot",
		Icon:     "✂️", Keywords: []string{"snip", "screenshot", "capture", "snipping"},
	},
	{
		ID: "cmd", Name: "Command Prompt",
		Subtitle: "Open Command Prompt",
		Icon:     "⌨️", Keywords: []string{"cmd", "command", "prompt", "terminal", "console"},
	},
	{
		ID: "powershell", Name: "PowerShell",
		Subtitle: "Open PowerShell",
		Icon:     "💠", Keywords: []string{"powershell", "ps", "terminal", "shell"},
	},
	{
		ID: "run-dialog", Name: "Run Dialog",
		Subtitle: "Open the Run dialog (Win+R)",
		Icon:     "▶️", Keywords: []string{"run", "dialog", "execute", "winr"},
	},
	{
		ID: "recycle-bin", Name: "Empty Recycle Bin",
		Subtitle: "Permanently delete recycled files",
		Icon:     "🗑️", Keywords: []string{"recycle", "bin", "trash", "empty", "delete"},
	},

	// --- Windows Settings (ms-settings: URIs) ---
	{
		ID: "settings-home", Name: "Settings",
		Subtitle: "Open Windows Settings",
		Icon:     "⚙️", Keywords: []string{"settings", "options", "preferences", "control panel"},
	},
	{
		ID: "settings-display", Name: "Display Settings",
		Subtitle: "Resolution, brightness, night light",
		Icon:     "🖥️", Keywords: []string{"display", "screen", "resolution", "brightness", "monitor"},
	},
	{
		ID: "settings-sound", Name: "Sound Settings",
		Subtitle: "Volume, output, input devices",
		Icon:     "🔊", Keywords: []string{"sound", "audio", "volume", "speaker", "microphone"},
	},
	{
		ID: "settings-bluetooth", Name: "Bluetooth Settings",
		Subtitle: "Pair and manage Bluetooth devices",
		Icon:     "📶", Keywords: []string{"bluetooth", "pair", "wireless", "headset"},
	},
	{
		ID: "settings-wifi", Name: "Wi-Fi Settings",
		Subtitle: "Connect to networks",
		Icon:     "📡", Keywords: []string{"wifi", "wi-fi", "network", "internet", "wireless"},
	},
	{
		ID: "settings-network", Name: "Network Settings",
		Subtitle: "Ethernet, VPN, proxy settings",
		Icon:     "🌐", Keywords: []string{"network", "ethernet", "vpn", "proxy", "internet"},
	},
	{
		ID: "settings-updates", Name: "Windows Update",
		Subtitle: "Check for Windows updates",
		Icon:     "🔁", Keywords: []string{"update", "windows update", "patch", "upgrade"},
	},
	{
		ID: "settings-apps", Name: "Apps & Features",
		Subtitle: "Install, uninstall, manage apps",
		Icon:     "📦", Keywords: []string{"apps", "features", "uninstall", "programs", "install"},
	},
	{
		ID: "settings-startup", Name: "Startup Apps",
		Subtitle: "Manage apps that run at startup",
		Icon:     "🚀", Keywords: []string{"startup", "autostart", "boot", "login"},
	},
	{
		ID: "settings-privacy", Name: "Privacy Settings",
		Subtitle: "Location, camera, microphone permissions",
		Icon:     "🛡️", Keywords: []string{"privacy", "location", "camera", "microphone", "permissions"},
	},
	{
		ID: "settings-accounts", Name: "Accounts Settings",
		Subtitle: "Your account, sign-in options",
		Icon:     "👤", Keywords: []string{"account", "user", "profile", "password", "pin", "signin"},
	},
	{
		ID: "settings-storage", Name: "Storage Settings",
		Subtitle: "Storage usage, cleanup",
		Icon:     "💿", Keywords: []string{"storage", "disk", "space", "cleanup", "drive"},
	},
	{
		ID: "settings-accessibility", Name: "Accessibility Settings",
		Subtitle: "Vision, hearing, interaction options",
		Icon:     "♿", Keywords: []string{"accessibility", "ease of access", "vision", "hearing"},
	},
	{
		ID: "settings-datetime", Name: "Date & Time Settings",
		Subtitle: "Time zone, clock format",
		Icon:     "🕐", Keywords: []string{"date", "time", "timezone", "clock"},
	},
	{
		ID: "settings-language", Name: "Language & Region",
		Subtitle: "Language, region, keyboard",
		Icon:     "🌍", Keywords: []string{"language", "region", "locale", "keyboard", "input"},
	},
	{
		ID: "settings-mouse", Name: "Mouse Settings",
		Subtitle: "Pointer speed, buttons, scroll",
		Icon:     "🖱️", Keywords: []string{"mouse", "pointer", "cursor", "scroll"},
	},
	{
		ID: "settings-notifications", Name: "Notifications Settings",
		Subtitle: "App notifications, focus assist",
		Icon:     "🔔", Keywords: []string{"notifications", "alerts", "focus", "do not disturb"},
	},
	{
		ID: "settings-power", Name: "Power & Sleep Settings",
		Subtitle: "Screen timeout, sleep settings",
		Icon:     "⚡", Keywords: []string{"power", "sleep", "battery", "screen timeout", "energy"},
	},
	{
		ID: "settings-personalization", Name: "Personalization",
		Subtitle: "Background, colors, themes",
		Icon:     "🎨", Keywords: []string{"personalize", "wallpaper", "background", "theme", "color", "dark mode"},
	},
	{
		ID: "settings-taskbar", Name: "Taskbar Settings",
		Subtitle: "Configure the taskbar",
		Icon:     "📌", Keywords: []string{"taskbar", "pinned", "start menu"},
	},
}

func ExecuteSystemCommand(id string) error {
	switch id {
	// Power
	case "lock-screen":
		return runHidden("rundll32.exe", "user32.dll,LockWorkStation")
	case "sleep":
		return runHidden("rundll32.exe", "powrprof.dll,SetSuspendState", "0", "1", "0")
	case "shutdown":
		return runHidden("shutdown.exe", "/s", "/t", "0")
	case "restart":
		return runHidden("shutdown.exe", "/r", "/t", "0")
	case "logout":
		return runHidden("shutdown.exe", "/l")
	case "hibernate":
		return runHidden("shutdown.exe", "/h")
	case "recycle-bin":
		return runHidden("powershell.exe", "-NoProfile", "-Command",
			"Clear-RecycleBin -Force -ErrorAction SilentlyContinue")

	// System Apps
	case "task-manager":
		return startHidden("taskmgr.exe")
	case "file-explorer":
		return startHidden("explorer.exe")
	case "calculator":
		return startHidden("calc.exe")
	case "notepad":
		return startHidden("notepad.exe")
	case "snipping-tool":
		return startMS("ms-screenclip:")
	case "cmd":
		return startHidden("cmd.exe")
	case "powershell":
		return startHidden("powershell.exe")
	case "run-dialog":
		return runHidden("cmd.exe", "/c", "start", "", "shell:AppsFolder\\Microsoft.Windows.Run_cw5n1h2txyewy!Run")

	// Settings
	case "settings-home":
		return startMS("ms-settings:")
	case "settings-display":
		return startMS("ms-settings:display")
	case "settings-sound":
		return startMS("ms-settings:sound")
	case "settings-bluetooth":
		return startMS("ms-settings:bluetooth")
	case "settings-wifi":
		return startMS("ms-settings:network-wifi")
	case "settings-network":
		return startMS("ms-settings:network-status")
	case "settings-updates":
		return startMS("ms-settings:windowsupdate")
	case "settings-apps":
		return startMS("ms-settings:appsfeatures")
	case "settings-startup":
		return startMS("ms-settings:startupapps")
	case "settings-privacy":
		return startMS("ms-settings:privacy")
	case "settings-accounts":
		return startMS("ms-settings:accounts")
	case "settings-storage":
		return startMS("ms-settings:storagesense")
	case "settings-accessibility":
		return startMS("ms-settings:easeofaccess-display")
	case "settings-datetime":
		return startMS("ms-settings:dateandtime")
	case "settings-language":
		return startMS("ms-settings:regionlanguage")
	case "settings-mouse":
		return startMS("ms-settings:mousetouchpad")
	case "settings-notifications":
		return startMS("ms-settings:notifications")
	case "settings-power":
		return startMS("ms-settings:powersleep")
	case "settings-personalization":
		return startMS("ms-settings:personalization")
	case "settings-taskbar":
		return startMS("ms-settings:taskbar")
	}
	return nil
}

// startMS opens a ms-settings: URI via cmd start.
func startMS(uri string) error {
	cmd := exec.Command("cmd.exe", "/c", "start", "", uri)
	cmd.SysProcAttr = hiddenSysProcAttr()
	return cmd.Start()
}

// startHidden starts a process without a visible console window.
func startHidden(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = hiddenSysProcAttr()
	return cmd.Start()
}

func runHidden(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = hiddenSysProcAttr()
	return cmd.Run()
}
