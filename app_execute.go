package main

import (
	"fmt"
	"net/url"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"

	"blight/internal/apps"
	"blight/internal/commands"
	"blight/internal/debug"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) Execute(id string) string {
	debug.Get().Info("execute", map[string]interface{}{"id": id})

	if strings.HasPrefix(id, "web-search:") {
		query := strings.TrimPrefix(id, "web-search:")
		tmpl := a.config.SearchEngineURL
		if tmpl == "" {
			tmpl = "https://www.google.com/search?q=%s"
		}
		searchURL := strings.ReplaceAll(tmpl, "%s", url.QueryEscape(query))
		runtime.BrowserOpenURL(a.ctx, searchURL)
		runtime.WindowHide(a.ctx)
		a.visible.Store(false)
		return "ok"
	}

	if strings.HasPrefix(id, "url-open:") {
		target := strings.TrimPrefix(id, "url-open:")
		runtime.BrowserOpenURL(a.ctx, target)
		runtime.WindowHide(a.ctx)
		a.visible.Store(false)
		return "ok"
	}

	if strings.HasPrefix(id, "cmd-needs-arg:") {
		return "needs-arg"
	}

	if strings.HasPrefix(id, "cmd-url:") {
		a.usage.Record(id)
		target := strings.TrimPrefix(id, "cmd-url:")
		runtime.BrowserOpenURL(a.ctx, target)
		runtime.WindowHide(a.ctx)
		a.visible.Store(false)
		return "ok"
	}

	if strings.HasPrefix(id, "cmd-copy:") {
		a.usage.Record(id)
		text := strings.TrimPrefix(id, "cmd-copy:")
		runtime.ClipboardSetText(a.ctx, text)
		return "copied"
	}

	if strings.HasPrefix(id, "cmd-path:") {
		a.usage.Record(id)
		path := strings.TrimPrefix(id, "cmd-path:")
		shellOpen(path)
		runtime.WindowHide(a.ctx)
		a.visible.Store(false)
		return "ok"
	}

	if strings.HasPrefix(id, "cmd-shell:") {
		a.usage.Record(id)
		cmd := strings.TrimPrefix(id, "cmd-shell:")
		var c *exec.Cmd
		if goruntime.GOOS == "windows" {
			c = exec.Command("cmd.exe", "/c", cmd)
		} else {
			c = exec.Command("sh", "-c", cmd)
		}
		configureSettingsCommand(c)
		if err := c.Start(); err != nil {
			return err.Error()
		}
		runtime.WindowHide(a.ctx)
		a.visible.Store(false)
		return "ok"
	}

	if strings.HasPrefix(id, "alias:") {
		trigger := strings.TrimPrefix(id, "alias:")
		expansion, ok := a.config.Aliases[trigger]
		if !ok {
			return "not found"
		}
		if strings.HasPrefix(expansion, "http://") || strings.HasPrefix(expansion, "https://") {
			runtime.BrowserOpenURL(a.ctx, expansion)
			runtime.WindowHide(a.ctx)
			a.visible.Store(false)
			return "ok"
		}
		runtime.ClipboardSetText(a.ctx, expansion)
		return "copied"
	}

	if id == "calc-result" {
		runtime.ClipboardSetText(a.ctx, "")
		return "copied"
	}

	if strings.HasPrefix(id, "clip-") {
		idxStr := strings.TrimPrefix(id, "clip-")
		var idx int
		fmt.Sscanf(idxStr, "%d", &idx)
		if a.clipboard.CopyToClipboard(idx) {
			return "copied"
		}
		return "error"
	}

	if strings.HasPrefix(id, "sys-") {
		a.usage.Record(id)
		sysID := strings.TrimPrefix(id, "sys-")
		if err := commands.ExecuteSystemCommand(sysID); err != nil {
			return err.Error()
		}
		return "ok"
	}

	if strings.HasPrefix(id, "file-open:") {
		filePath := strings.TrimPrefix(id, "file-open:")
		a.usage.Record("file-open:" + filePath)
		shellOpen(filePath)
		runtime.WindowHide(a.ctx)
		a.visible.Store(false)
		return "ok"
	}

	if strings.HasPrefix(id, "file-reveal:") {
		filePath := strings.TrimPrefix(id, "file-reveal:")
		explorerSelect(filePath)
		return "ok"
	}

	if strings.HasPrefix(id, "dir-open:") {
		dirPath := strings.TrimPrefix(id, "dir-open:")
		a.usage.Record("dir-open:" + dirPath)
		shellOpen(dirPath)
		runtime.WindowHide(a.ctx)
		a.visible.Store(false)
		return "ok"
	}

	allApps := a.scanner.Apps()
	for _, app := range allApps {
		if app.Name == id {
			a.usage.Record(id)
			if err := apps.Launch(app); err != nil {
				return err.Error()
			}
			runtime.WindowHide(a.ctx)
			return "ok"
		}
	}
	return "not found"
}

func (a *App) GetContextActions(id string) []ContextAction {
	switch {
	case strings.HasPrefix(id, "dir-open:"):
		return []ContextAction{
			{ID: "open", Label: "Open", Icon: icon("\uE768", "▶"), Shortcut: "↵"},
			{ID: "terminal", Label: "Open in Terminal", Icon: icon("\uE756", "⌨"), Shortcut: "⌃↵"},
			{ID: "copy-path", Label: "Copy Path", Icon: icon("\uE8C8", "📋")},
		}
	case strings.HasPrefix(id, "file-open:"):
		return []ContextAction{
			{ID: "open", Label: "Open", Icon: icon("\uE768", "▶"), Shortcut: "↵"},
			{ID: "explorer", Label: revealLabel(), Icon: icon("\uE8B7", "📂"), Shortcut: "⌃↵"},
			{ID: "copy-path", Label: "Copy Path", Icon: icon("\uE8C8", "📋")},
			{ID: "copy-name", Label: "Copy Name", Icon: icon("\uE70F", "📝")},
		}
	case strings.HasPrefix(id, "clip-"):
		return []ContextAction{
			{ID: "copy", Label: "Copy", Icon: icon("\uE8C8", "📋"), Shortcut: "↵"},
			{ID: "delete", Label: "Delete", Icon: icon("\uE74D", "🗑️"), Destructive: true},
		}
	case strings.HasPrefix(id, "sys-"):
		return []ContextAction{
			{ID: "run", Label: "Run", Icon: icon("\uE768", "▶"), Shortcut: "↵"},
		}
	case strings.HasPrefix(id, "cmd-url:") || strings.HasPrefix(id, "cmd-copy:") || strings.HasPrefix(id, "cmd-path:") || strings.HasPrefix(id, "cmd-shell:"):
		return []ContextAction{
			{ID: "run", Label: "Run", Icon: icon("\uE768", "▶"), Shortcut: "↵"},
			{ID: "copy", Label: "Copy Value", Icon: icon("\uE8C8", "📋"), Shortcut: "⌃↵"},
		}
	case strings.HasPrefix(id, "cmd-needs-arg:"):
		return []ContextAction{}
	case strings.HasPrefix(id, "alias:"):
		return []ContextAction{
			{ID: "open", Label: "Use", Icon: icon("\uE768", "▶"), Shortcut: "↵"},
			{ID: "copy", Label: "Copy Expansion", Icon: icon("\uE8C8", "📋"), Shortcut: "⌃↵"},
			{ID: "delete-alias", Label: "Delete Alias", Icon: icon("\uE74D", "🗑️"), Destructive: true},
		}
	case id == "calc-result" || id == "no-results" || strings.HasPrefix(id, "web-search:"):
		return []ContextAction{}
	default:
		pinLabel := "Pin to Top"
		pinIcon := icon("\uE718", "📌")
		for _, p := range a.config.PinnedItems {
			if p == id {
				pinLabel = "Unpin from Top"
				pinIcon = icon("\uE77A", "📌")
				break
			}
		}
		return []ContextAction{
			{ID: "open", Label: "Open", Icon: icon("\uE768", "▶"), Shortcut: "↵"},
			{ID: "admin", Label: elevateLabel(), Icon: icon("\uE7EF", "🛡️"), Shortcut: "⌃↵"},
			{ID: "explorer", Label: revealLabel(), Icon: icon("\uE8B7", "📂")},
			{ID: "copy-path", Label: "Copy Path", Icon: icon("\uE8C8", "📋")},
			{ID: "pin", Label: pinLabel, Icon: pinIcon},
		}
	}
}

func (a *App) ExecuteContextAction(resultID string, actionID string) string {
	if strings.HasPrefix(resultID, "cmd-url:") || strings.HasPrefix(resultID, "cmd-copy:") ||
		strings.HasPrefix(resultID, "cmd-path:") || strings.HasPrefix(resultID, "cmd-shell:") {
		switch actionID {
		case "run":
			return a.Execute(resultID)
		case "copy":
			parts := strings.SplitN(resultID, ":", 2)
			if len(parts) == 2 {
				runtime.ClipboardSetText(a.ctx, parts[1])
				return "copied"
			}
		}
		return "unknown action"
	}

	if strings.HasPrefix(resultID, "alias:") {
		trigger := strings.TrimPrefix(resultID, "alias:")
		expansion, ok := a.config.Aliases[trigger]
		if !ok {
			return "not found"
		}
		switch actionID {
		case "open":
			if strings.HasPrefix(expansion, "http://") || strings.HasPrefix(expansion, "https://") {
				runtime.BrowserOpenURL(a.ctx, expansion)
				runtime.WindowHide(a.ctx)
				a.visible.Store(false)
			} else {
				runtime.ClipboardSetText(a.ctx, expansion)
			}
			return "ok"
		case "copy":
			runtime.ClipboardSetText(a.ctx, expansion)
			return "ok"
		case "delete-alias":
			delete(a.config.Aliases, trigger)
			_ = a.saveConfig()
			return "ok"
		}
		return "unknown action"
	}

	if strings.HasPrefix(resultID, "dir-open:") {
		dirPath := strings.TrimPrefix(resultID, "dir-open:")
		switch actionID {
		case "open":
			shellOpen(dirPath)
			runtime.WindowHide(a.ctx)
			a.visible.Store(false)
			return "ok"
		case "terminal":
			openInTerminal(dirPath)
			return "ok"
		case "copy-path":
			runtime.ClipboardSetText(a.ctx, dirPath)
			return "ok"
		}
		return "unknown action"
	}

	if strings.HasPrefix(resultID, "file-open:") {
		filePath := strings.TrimPrefix(resultID, "file-open:")
		switch actionID {
		case "open":
			shellOpen(filePath)
			runtime.WindowHide(a.ctx)
			a.visible.Store(false)
			return "ok"
		case "explorer":
			explorerSelect(filePath)
			return "ok"
		case "copy-path":
			runtime.ClipboardSetText(a.ctx, filePath)
			return "ok"
		case "copy-name":
			runtime.ClipboardSetText(a.ctx, filepath.Base(filePath))
			return "ok"
		}
		return "unknown action"
	}

	if strings.HasPrefix(resultID, "clip-") {
		idxStr := strings.TrimPrefix(resultID, "clip-")
		var idx int
		fmt.Sscanf(idxStr, "%d", &idx)
		switch actionID {
		case "copy", "open":
			if a.clipboard.CopyToClipboard(idx) {
				return "copied"
			}
			return "error"
		case "delete":
			a.clipboard.Delete(idx)
			return "ok"
		}
		return "unknown action"
	}

	if strings.HasPrefix(resultID, "sys-") {
		if actionID == "run" {
			sysID := strings.TrimPrefix(resultID, "sys-")
			if err := commands.ExecuteSystemCommand(sysID); err != nil {
				return err.Error()
			}
			return "ok"
		}
		return "unknown action"
	}

	allApps := a.scanner.Apps()
	var target apps.AppEntry
	found := false
	for _, app := range allApps {
		if app.Name == resultID {
			target = app
			found = true
			break
		}
	}
	if !found {
		return "not found"
	}

	switch actionID {
	case "open":
		a.usage.Record(resultID)
		if err := apps.Launch(target); err != nil {
			return err.Error()
		}
		runtime.WindowHide(a.ctx)
		return "ok"
	case "admin":
		a.usage.Record(resultID)
		if err := runAsAdmin(target.Path); err != nil {
			return err.Error()
		}
		runtime.WindowHide(a.ctx)
		return "ok"
	case "explorer":
		explorerSelect(target.Path)
		return "ok"
	case "copy-path":
		runtime.ClipboardSetText(a.ctx, target.Path)
		return "ok"
	case "pin":
		if a.TogglePinned(resultID) {
			return "pinned"
		}
		return "unpinned"
	}

	return "unknown action"
}

// EvalCalc evaluates a simple arithmetic expression and returns the result as a
// string, or an empty string on error.
func (a *App) EvalCalc(expr string) string {
	if !commands.IsCalcQuery(expr) {
		return ""
	}
	result := commands.Evaluate(expr)
	if !result.Valid {
		return ""
	}
	return result.Result
}
