package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"blight/internal/debug"
	"blight/internal/files"
	"blight/internal/hotkey"
	"blight/internal/startup"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".blight")
}

func (a *App) configPath() string {
	return filepath.Join(a.configDir(), "config.json")
}

func defaultConfig() BlightConfig {
	return BlightConfig{
		FirstRun:            true,
		Hotkey:              "Alt+Space",
		MaxClipboard:        50,
		MaxResults:          8,
		SearchDelay:         120,
		HideWhenDeactivated: true,
		LastQueryMode:       "clear",
		WindowPosition:      "center",
		UseAnimation:        true,
		ShowPlaceholder:     true,
		PlaceholderText:     "",
		Theme:               "dark",
		FooterHints:         "always",
		StartOnStartup:      false,
		HideNotifyIcon:      false,
	}
}

func (a *App) loadConfig() {
	data, err := os.ReadFile(a.configPath())
	if err != nil {
		a.config = defaultConfig()
		return
	}
	// Start with defaults so new fields are initialised even on old config files
	a.config = defaultConfig()
	if err := json.Unmarshal(data, &a.config); err != nil {
		a.config = defaultConfig()
		return
	}
	// Clamp / validate
	if a.config.MaxClipboard == 0 {
		a.config.MaxClipboard = 50
	}
	if a.config.MaxResults == 0 {
		a.config.MaxResults = 8
	}
	if a.config.SearchDelay == 0 {
		a.config.SearchDelay = 120
	}
	if a.config.LastQueryMode == "" {
		a.config.LastQueryMode = "clear"
	}
	if a.config.WindowPosition == "" {
		a.config.WindowPosition = "center"
	}
	if a.config.Theme == "" {
		a.config.Theme = "dark"
	}
	if a.config.FooterHints == "" {
		a.config.FooterHints = "always"
	}

	// Migrate aliases → CommandDefinitions (idempotent: skip keywords already present)
	if len(a.config.Aliases) > 0 {
		cmdKeywords := make(map[string]bool)
		for _, c := range a.config.Commands {
			cmdKeywords[strings.ToLower(c.Keyword)] = true
		}
		for trigger, expansion := range a.config.Aliases {
			t := strings.ToLower(trigger)
			if cmdKeywords[t] {
				continue
			}
			actionType := "copy_text"
			if strings.HasPrefix(expansion, "http://") || strings.HasPrefix(expansion, "https://") {
				actionType = "open_url"
			}
			a.config.Commands = append(a.config.Commands, CommandDefinition{
				ID:               "alias-" + t,
				Title:            trigger,
				Keyword:          t,
				Description:      expansion,
				ActionType:       actionType,
				Template:         expansion,
				RequiresArgument: strings.Contains(expansion, "{{query}}"),
			})
			cmdKeywords[t] = true
		}
	}
}

func (a *App) saveConfig() error {
	if err := os.MkdirAll(a.configDir(), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(a.config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(a.configPath(), data, 0644)
}

func (a *App) IsFirstRun() bool {
	return a.config.FirstRun
}

func (a *App) CompleteOnboarding(hk, theme string, useAnimation bool) error {
	a.config.FirstRun = false
	if hk != "" {
		a.config.Hotkey = hk
	}
	if theme != "" {
		a.config.Theme = theme
	}
	a.config.UseAnimation = useAnimation
	return a.saveConfig()
}

// GetConfig returns the current config for the settings UI.
func (a *App) GetConfig() BlightConfig {
	return a.config
}

// SaveSettings persists settings from the settings UI.
// cfg is a partial BlightConfig — only non-zero/non-empty fields overwrite the
// current config so the frontend can send only the fields it knows about.
func (a *App) SaveSettings(cfg BlightConfig) error {
	log := debug.Get()

	if cfg.Hotkey != "" {
		prev := a.config.Hotkey
		a.config.Hotkey = cfg.Hotkey
		if a.hotkey != nil && cfg.Hotkey != prev {
			a.hotkey.Stop()
			a.hotkey = hotkey.New(cfg.Hotkey, func() { a.ToggleWindow() })
			if err := a.hotkey.Start(); err != nil {
				log.Error("hotkey restart failed", map[string]interface{}{"error": err.Error()})
			} else {
				log.Info("global hotkey updated", map[string]interface{}{"hotkey": cfg.Hotkey})
			}
		}
	}
	if cfg.MaxClipboard > 0 {
		a.config.MaxClipboard = cfg.MaxClipboard
		if a.clipboard != nil {
			a.clipboard.SetMaxSize(cfg.MaxClipboard)
		}
	}
	if cfg.IndexDirs != nil {
		dirsChanged := !stringSlicesEqual(a.config.IndexDirs, cfg.IndexDirs)
		a.config.IndexDirs = cfg.IndexDirs
		if a.fileIdx != nil {
			a.fileIdx.UpdateDirs(cfg.IndexDirs)
			if dirsChanged {
				go a.fileIdx.Reindex()
			}
		}
	}
	if cfg.MaxResults > 0 {
		a.config.MaxResults = cfg.MaxResults
	}
	if cfg.SearchDelay > 0 {
		a.config.SearchDelay = cfg.SearchDelay
	}
	if cfg.LastQueryMode != "" {
		a.config.LastQueryMode = cfg.LastQueryMode
	}
	if cfg.WindowPosition != "" {
		a.config.WindowPosition = cfg.WindowPosition
	}
	if cfg.Theme != "" {
		a.config.Theme = cfg.Theme
	}
	if cfg.FooterHints != "" {
		a.config.FooterHints = cfg.FooterHints
	}
	if cfg.PlaceholderText != "" {
		a.config.PlaceholderText = cfg.PlaceholderText
	}
	// Boolean fields are always updated (they can legitimately be false)
	a.config.HideWhenDeactivated = cfg.HideWhenDeactivated
	a.config.UseAnimation = cfg.UseAnimation
	a.config.ShowPlaceholder = cfg.ShowPlaceholder
	a.config.HideNotifyIcon = cfg.HideNotifyIcon
	a.config.DisableFolderIndex = cfg.DisableFolderIndex

	if cfg.Aliases != nil {
		a.config.Aliases = cfg.Aliases
	}
	if cfg.PinnedItems != nil {
		a.config.PinnedItems = cfg.PinnedItems
	}

	// System startup: sync Windows registry
	if cfg.StartOnStartup != a.config.StartOnStartup {
		a.config.StartOnStartup = cfg.StartOnStartup
		if cfg.StartOnStartup {
			if err := startup.Enable(); err != nil {
				log.Error("startup.Enable failed", map[string]interface{}{"error": err.Error()})
			}
		} else {
			if err := startup.Disable(); err != nil {
				log.Error("startup.Disable failed", map[string]interface{}{"error": err.Error()})
			}
		}
	}

	// Tray icon visibility
	if a.tray != nil {
		if a.config.HideNotifyIcon {
			a.tray.Stop()
		} else {
			a.tray.Start()
		}
	}

	return a.saveConfig()
}

// GetStartupEnabled returns whether blight is currently registered to start on login.
func (a *App) GetStartupEnabled() bool {
	return startup.IsEnabled()
}

func (a *App) OpenFolderPicker() string {
	path, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Directory to Index",
	})
	if err != nil {
		return ""
	}
	return path
}

// ExportSettings returns the current config as a JSON string.
func (a *App) ExportSettings() string {
	data, err := json.MarshalIndent(a.config, "", "  ")
	if err != nil {
		return ""
	}
	return string(data)
}

// ImportSettings replaces the current config with one parsed from a JSON string.
func (a *App) ImportSettings(data string) error {
	var cfg BlightConfig
	if err := json.Unmarshal([]byte(data), &cfg); err != nil {
		return fmt.Errorf("invalid settings JSON: %w", err)
	}
	a.config = cfg
	return a.saveConfig()
}

// GetUsageScores returns a map of item ID → decayed usage score for items with
// at least one recorded use. Used by the frontend to show frequency indicators.
func (a *App) GetUsageScores() map[string]int {
	scores := make(map[string]int)
	for _, app := range a.scanner.Apps() {
		if s := a.usage.Score(app.Name); s > 0 {
			scores[app.Name] = s
		}
	}
	return scores
}

// GetAliases returns the current alias map (trigger → expansion).
func (a *App) GetAliases() map[string]string {
	if a.config.Aliases == nil {
		return map[string]string{}
	}
	return a.config.Aliases
}

// SaveAlias creates or updates an alias.
func (a *App) SaveAlias(trigger, expansion string) error {
	trigger = strings.TrimSpace(trigger)
	expansion = strings.TrimSpace(expansion)
	if trigger == "" || expansion == "" {
		return fmt.Errorf("trigger and expansion must not be empty")
	}
	if a.config.Aliases == nil {
		a.config.Aliases = make(map[string]string)
	}
	a.config.Aliases[strings.ToLower(trigger)] = expansion
	return a.saveConfig()
}

// DeleteAlias removes an alias by trigger.
func (a *App) DeleteAlias(trigger string) error {
	if a.config.Aliases != nil {
		delete(a.config.Aliases, trigger)
	}
	return a.saveConfig()
}

// GetCommands returns all user-defined commands.
func (a *App) GetCommands() []CommandDefinition {
	if a.config.Commands == nil {
		return []CommandDefinition{}
	}
	return a.config.Commands
}

// SaveCommand creates or updates a command by ID.
func (a *App) SaveCommand(cmd CommandDefinition) error {
	cmd.ID = strings.TrimSpace(cmd.ID)
	cmd.Keyword = strings.ToLower(strings.TrimSpace(cmd.Keyword))
	if cmd.ID == "" || cmd.Keyword == "" || cmd.Template == "" {
		return fmt.Errorf("id, keyword, and template are required")
	}
	if cmd.ActionType == "" {
		cmd.ActionType = "open_url"
	}
	for i, c := range a.config.Commands {
		if c.ID == cmd.ID {
			a.config.Commands[i] = cmd
			return a.saveConfig()
		}
	}
	a.config.Commands = append(a.config.Commands, cmd)
	return a.saveConfig()
}

// DeleteCommand removes a command by ID.
func (a *App) DeleteCommand(id string) error {
	for i, c := range a.config.Commands {
		if c.ID == id {
			a.config.Commands = append(a.config.Commands[:i], a.config.Commands[i+1:]...)
			return a.saveConfig()
		}
	}
	return nil
}

// TogglePinned pins an item if not already pinned, or unpins it if it is.
// Returns true if the item is now pinned, false if it was unpinned.
func (a *App) TogglePinned(id string) bool {
	for i, p := range a.config.PinnedItems {
		if p == id {
			a.config.PinnedItems = append(a.config.PinnedItems[:i], a.config.PinnedItems[i+1:]...)
			_ = a.saveConfig()
			return false
		}
	}
	a.config.PinnedItems = append(a.config.PinnedItems, id)
	_ = a.saveConfig()
	return true
}

func (a *App) ReindexFiles() {
	a.fileIdx.Reindex()
}

func (a *App) CancelIndex() {
	a.fileIdx.CancelIndex()
}

func (a *App) ClearIndex() {
	a.fileIdx.ClearIndex()
}

func (a *App) GetIndexStatus() files.IndexStatus {
	return a.fileIdx.Status()
}
