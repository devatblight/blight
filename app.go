package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"time"

	"blight/internal/apps"
	"blight/internal/commands"
	"blight/internal/debug"
	"blight/internal/files"
	"blight/internal/hotkey"
	"blight/internal/search"
	"blight/internal/tray"
	"blight/internal/updater"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type UpdateInfo struct {
	Available bool   `json:"available"`
	Version   string `json:"version"`
	URL       string `json:"url"`
	Notes     string `json:"notes"`
	Error     string `json:"error,omitempty"`
}

type SearchResult struct {
	ID                   string `json:"id"`
	Title                string `json:"title"`
	Subtitle             string `json:"subtitle"`
	Icon                 string `json:"icon"`
	Category             string `json:"category"`
	Path                 string `json:"path"`
	Kind                 string `json:"kind"`
	PrimaryActionLabel   string `json:"primaryActionLabel"`
	SecondaryActionLabel string `json:"secondaryActionLabel,omitempty"`
	SupportsActions      bool   `json:"supportsActions"`
}

type ContextAction struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Icon        string `json:"icon"`
	Shortcut    string `json:"shortcut,omitempty"`
	Destructive bool   `json:"destructive,omitempty"`
}

type CommandDefinition struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	Keyword          string   `json:"keyword"`
	Description      string   `json:"description"`
	ActionType       string   `json:"actionType"` // open_url|copy_text|open_path|run_shell
	Template         string   `json:"template"`
	Keywords         []string `json:"keywords,omitempty"`
	Icon             string   `json:"icon,omitempty"`
	RequiresArgument bool     `json:"requiresArgument"`
	RunAsAdmin       bool     `json:"runAsAdmin"`
	Pinned           bool     `json:"pinned"`
}

type BlightConfig struct {
	// Core
	FirstRun     bool     `json:"firstRun"`
	Hotkey       string   `json:"hotkey"`
	MaxClipboard int      `json:"maxClipboard"`
	IndexDirs    []string `json:"indexDirs,omitempty"`

	// Search behaviour
	MaxResults  int `json:"maxResults"`
	SearchDelay int `json:"searchDelay"`

	// Window behaviour
	HideWhenDeactivated bool   `json:"hideWhenDeactivated"`
	LastQueryMode       string `json:"lastQueryMode"`
	WindowPosition      string `json:"windowPosition"`

	// Appearance
	UseAnimation    bool   `json:"useAnimation"`
	ShowPlaceholder bool   `json:"showPlaceholder"`
	PlaceholderText string `json:"placeholderText"`
	Theme           string `json:"theme"`
	FooterHints     string `json:"footerHints"`

	// System integration
	StartOnStartup bool      `json:"startOnStartup"`
	HideNotifyIcon bool      `json:"hideNotifyIcon"`
	LastIndexedAt  time.Time `json:"lastIndexedAt,omitempty"`

	// File index behaviour
	DisableFolderIndex bool `json:"disableFolderIndex,omitempty"`

	// Web search
	SearchEngineURL string `json:"searchEngineURL,omitempty"`

	// User-defined aliases and commands
	Aliases     map[string]string   `json:"aliases,omitempty"`
	Commands    []CommandDefinition `json:"commands,omitempty"`
	PinnedItems []string            `json:"pinnedItems,omitempty"`
}

type App struct {
	ctx          context.Context
	config       BlightConfig
	scanner      *apps.Scanner
	usage        *search.UsageTracker
	clipboard    *commands.ClipboardHistory
	fileIdx      *files.FileIndex
	hotkey       *hotkey.HotkeyManager
	tray         *tray.TrayIcon
	visible      atomic.Bool
	lastShownAt  atomic.Int64
	version      string
	settingsMode bool
}

func NewApp(version string) *App {
	return &App{version: version}
}

func NewSettingsApp(version string) *App {
	return &App{version: version, settingsMode: true}
}

func (a *App) startup(ctx context.Context) {
	if a.settingsMode {
		a.settingsOnlyStartup(ctx)
		return
	}

	log := debug.Get()
	defer log.RecoverPanic("app.startup")

	log.Info("app.startup called", map[string]interface{}{"version": a.version})
	a.ctx = ctx
	a.visible.Store(true)
	a.loadConfig()
	log.Debug("config loaded", map[string]interface{}{"firstRun": a.config.FirstRun, "hotkey": a.config.Hotkey})

	a.scanner = apps.NewScanner()
	log.Info("app scanner initialized", map[string]interface{}{"appCount": len(a.scanner.Apps())})

	a.usage = search.NewUsageTracker()
	a.clipboard = commands.NewClipboardHistory(ctx)
	if a.config.MaxClipboard > 0 {
		a.clipboard.SetMaxSize(a.config.MaxClipboard)
	}
	go a.clipboard.PollClipboard()
	log.Debug("clipboard polling started")

	a.fileIdx = files.NewFileIndex(a.config.IndexDirs, func(status files.IndexStatus) {
		log.Debug("index status changed", map[string]interface{}{"state": status.State, "message": status.Message, "count": status.Count})
		runtime.EventsEmit(ctx, "indexStatus", status)
		if status.State == "ready" {
			a.config.LastIndexedAt = time.Now()
			_ = a.saveConfig()
		}
	})
	const staleAge = 72 * time.Hour
	if a.fileIdx.IsStale(staleAge) {
		a.fileIdx.Start()
	}
	log.Info("file indexer started")

	hotkeyStr := a.config.Hotkey
	if hotkeyStr == "" {
		hotkeyStr = "Alt+Space"
	}
	a.hotkey = hotkey.New(hotkeyStr, func() {
		log.Debug("hotkey triggered", map[string]interface{}{"hotkey": hotkeyStr})
		a.ToggleWindow()
	})
	a.hotkey.Start()
	log.Info("global hotkey registered", map[string]interface{}{"hotkey": hotkeyStr})

	a.tray = tray.New(
		func() { a.ShowWindow() },
		func() {
			log.Info("settings requested from tray")
			a.OpenSettingsWindow()
		},
		func() { runtime.Quit(ctx) },
	)
	a.tray.Start()
	log.Info("system tray icon created")

	log.Info("startup complete")
}

func (a *App) settingsOnlyStartup(ctx context.Context) {
	a.ctx = ctx
	a.loadConfig()
	runtime.EventsEmit(ctx, "openSettings")
}

func (a *App) settingsShutdown(_ context.Context) {}

func (a *App) CloseSettings() {
	runtime.Quit(a.ctx)
}

func (a *App) shutdown(ctx context.Context) {
	log := debug.Get()
	log.Info("shutdown called")
	if a.hotkey != nil {
		a.hotkey.Stop()
	}
	if a.tray != nil {
		a.tray.Stop()
	}
	log.Info("cleanup complete")
}

func (a *App) CheckForUpdates() UpdateInfo {
	u := updater.New("devatblight/blight")
	log := debug.Get()

	log.Info("checking for updates", map[string]interface{}{"current": a.version})

	rel, found, err := u.CheckForUpdates(a.version)
	if err != nil {
		log.Error("update check failed", map[string]interface{}{"error": err.Error()})
		return UpdateInfo{Error: err.Error()}
	}

	if !found {
		log.Info("no updates found")
		return UpdateInfo{Available: false}
	}

	log.Info("update available", map[string]interface{}{"version": rel.Version})

	return UpdateInfo{
		Available: true,
		Version:   rel.Version,
		URL:       rel.URL,
		Notes:     rel.Notes,
	}
}

func (a *App) InstallUpdate() string {
	log := debug.Get()
	u := updater.New("devatblight/blight")

	rel, found, err := u.CheckForUpdates(a.version)
	if err != nil {
		return "Check failed: " + err.Error()
	}
	if !found {
		return "No update found"
	}

	log.Info("installing update", map[string]interface{}{"version": rel.Version})
	err = u.ApplyUpdateWithProgress(rel, func(pct int) {
		runtime.EventsEmit(a.ctx, "updateProgress", pct)
	})
	if err != nil {
		log.Error("update failed", map[string]interface{}{"error": err.Error()})
		return "Update failed: " + err.Error()
	}

	log.Info("update applied — NSIS installer will handle kill and relaunch")
	return "success"
}

func (a *App) GetVersion() string {
	return a.version
}

func (a *App) IsSettingsMode() bool { return a.settingsMode }

func (a *App) GetDataDir() string {
	return a.configDir()
}

func (a *App) GetInstallDir() string {
	return blightInstallDir()
}

func (a *App) OpenFolder(path string) {
	shellOpen(path)
}

func (a *App) OpenURL(u string) {
	runtime.BrowserOpenURL(a.ctx, u)
}

func (a *App) Uninstall() string {
	dir := blightInstallDir()
	uninst := filepath.Join(dir, "uninstall.exe")
	if _, err := os.Stat(uninst); err != nil {
		return "not-found:" + uninst
	}
	cmd := exec.Command(uninst)
	if err := cmd.Start(); err != nil {
		return "error:" + err.Error()
	}
	return "success"
}

func (a *App) OpenSettingsWindow() {
	log := debug.Get()
	exe, err := os.Executable()
	if err != nil {
		log.Error("OpenSettingsWindow: could not get executable path", map[string]interface{}{"error": err.Error()})
		return
	}
	cmd := exec.Command(exe, "--settings")
	configureSettingsCommand(cmd)
	if err := cmd.Start(); err != nil {
		log.Error("OpenSettingsWindow: failed to spawn settings window", map[string]interface{}{"error": err.Error()})
	}
}

// spotlightHeight is the window height when no results are shown (search bar only).
const spotlightHeight = 72

func (a *App) ToggleWindow() {
	if a.visible.Load() {
		runtime.WindowHide(a.ctx)
		a.visible.Store(false)
	} else {
		a.loadConfig()
		a.lastShownAt.Store(time.Now().UnixNano())
		// Reset to compact height and re-centre so the search bar is always at
		// a predictable screen position; results will grow downward from there.
		a.resetWindowForShow()
		runtime.WindowShow(a.ctx)
		runtime.WindowSetAlwaysOnTop(a.ctx, true)
		runtime.EventsEmit(a.ctx, "windowShown")
		a.visible.Store(true)
	}
}

func (a *App) ShowWindow() {
	a.loadConfig()
	a.lastShownAt.Store(time.Now().UnixNano())
	a.resetWindowForShow()
	runtime.WindowShow(a.ctx)
	runtime.WindowSetAlwaysOnTop(a.ctx, true)
	runtime.EventsEmit(a.ctx, "windowShown")
	a.visible.Store(true)
}

// resetWindowForShow shrinks the window to the compact spotlight height and
// centres it on screen so the search bar always appears at the same position.
func (a *App) resetWindowForShow() {
	w, _ := runtime.WindowGetSize(a.ctx)
	runtime.WindowSetSize(a.ctx, w, spotlightHeight)
	runtime.WindowCenter(a.ctx)
}

// ResizeToContent adjusts the window height to h pixels.
// The top-left corner is NOT moved, so the window grows/shrinks from the
// bottom — the search bar stays at the same screen position.
// Called by the frontend after every render.
func (a *App) ResizeToContent(h int) {
	const minH, maxH = spotlightHeight, 680
	if h < minH {
		h = minH
	}
	if h > maxH {
		h = maxH
	}
	w, _ := runtime.WindowGetSize(a.ctx)
	runtime.WindowSetSize(a.ctx, w, h)
}


func (a *App) HideWindow() {
	const gracePeriod = 600 * time.Millisecond
	if time.Since(time.Unix(0, a.lastShownAt.Load())) < gracePeriod {
		return
	}
	runtime.WindowHide(a.ctx)
	a.visible.Store(false)
}

func (a *App) GetIcon(path string) string {
	return apps.GetIconBase64(path)
}

func (a *App) RefreshApps() {
	a.scanner.Scan()
}
