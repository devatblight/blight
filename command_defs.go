package main

import (
	"fmt"
	"net/url"
	"path/filepath"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type CommandDefinition struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	Keyword          string   `json:"keyword"`
	Description      string   `json:"description"`
	ActionType       string   `json:"actionType"`
	Template         string   `json:"template"`
	Keywords         []string `json:"keywords,omitempty"`
	Icon             string   `json:"icon,omitempty"`
	RequiresArgument bool     `json:"requiresArgument,omitempty"`
	RunAsAdmin       bool     `json:"runAsAdmin,omitempty"`
	Pinned           bool     `json:"pinned,omitempty"`
}

func builtinCommands() []CommandDefinition {
	return []CommandDefinition{
		{
			ID:               "builtin-google",
			Title:            "Google Search",
			Keyword:          "g",
			Description:      "Search Google for the typed query",
			ActionType:       "open_url",
			Template:         "https://www.google.com/search?q={{query}}",
			Keywords:         []string{"google", "search", "web"},
			RequiresArgument: true,
		},
		{
			ID:               "builtin-github",
			Title:            "GitHub Search",
			Keyword:          "gh",
			Description:      "Search GitHub repositories and code",
			ActionType:       "open_url",
			Template:         "https://github.com/search?q={{query}}",
			Keywords:         []string{"github", "code", "repo"},
			RequiresArgument: true,
		},
		{
			ID:               "builtin-youtube",
			Title:            "YouTube Search",
			Keyword:          "yt",
			Description:      "Search YouTube",
			ActionType:       "open_url",
			Template:         "https://www.youtube.com/results?search_query={{query}}",
			Keywords:         []string{"youtube", "video", "media"},
			RequiresArgument: true,
		},
		{
			ID:               "builtin-wikipedia",
			Title:            "Wikipedia Search",
			Keyword:          "wiki",
			Description:      "Search Wikipedia",
			ActionType:       "open_url",
			Template:         "https://en.wikipedia.org/w/index.php?search={{query}}",
			Keywords:         []string{"wikipedia", "encyclopedia", "docs"},
			RequiresArgument: true,
		},
		{
			ID:               "builtin-maps",
			Title:            "Maps Search",
			Keyword:          "maps",
			Description:      "Search a location in Google Maps",
			ActionType:       "open_url",
			Template:         "https://www.google.com/maps/search/{{query}}",
			Keywords:         []string{"maps", "location", "travel"},
			RequiresArgument: true,
		},
	}
}

func normalizeCommand(cmd CommandDefinition) (CommandDefinition, error) {
	cmd.ID = strings.TrimSpace(cmd.ID)
	cmd.Title = strings.TrimSpace(cmd.Title)
	cmd.Keyword = strings.ToLower(strings.TrimSpace(cmd.Keyword))
	cmd.Description = strings.TrimSpace(cmd.Description)
	cmd.ActionType = strings.TrimSpace(cmd.ActionType)
	cmd.Template = strings.TrimSpace(cmd.Template)

	if cmd.ID == "" {
		cmd.ID = "user-" + uuid.NewString()
	}
	if cmd.Title == "" {
		return CommandDefinition{}, fmt.Errorf("title must not be empty")
	}
	if cmd.Keyword == "" {
		return CommandDefinition{}, fmt.Errorf("keyword must not be empty")
	}

	switch cmd.ActionType {
	case "open_url", "copy_text", "open_path", "run_shell":
	default:
		return CommandDefinition{}, fmt.Errorf("unsupported action type: %s", cmd.ActionType)
	}

	if cmd.Template == "" {
		return CommandDefinition{}, fmt.Errorf("template must not be empty")
	}

	if cmd.ActionType == "open_url" &&
		!strings.Contains(cmd.Template, "{{query}}") &&
		!isURL(cmd.Template) {
		return CommandDefinition{}, fmt.Errorf("open_url templates must be an absolute URL or contain {{query}}")
	}

	for i, keyword := range cmd.Keywords {
		cmd.Keywords[i] = strings.ToLower(strings.TrimSpace(keyword))
	}

	return cmd, nil
}

func commandPinnedID(id string) string {
	return "command:" + id
}

func commandResultID(id, arg string) string {
	canonical := commandPinnedID(id)
	if arg == "" {
		return canonical
	}
	return canonical + "|" + url.QueryEscape(arg)
}

func parseCommandResultID(resultID string) (string, string, bool) {
	if !strings.HasPrefix(resultID, "command:") {
		return "", "", false
	}
	raw := strings.TrimPrefix(resultID, "command:")
	if idx := strings.Index(raw, "|"); idx >= 0 {
		id := raw[:idx]
		arg, _ := url.QueryUnescape(raw[idx+1:])
		return id, arg, true
	}
	return raw, "", true
}

func canonicalPinnedID(id string) string {
	if cmdID, _, ok := parseCommandResultID(id); ok {
		return commandPinnedID(cmdID)
	}
	return id
}

func commandSearchBlob(cmd CommandDefinition) []string {
	return []string{
		cmd.Title,
		cmd.Keyword,
		cmd.Description,
		cmd.Template,
		strings.Join(cmd.Keywords, " "),
	}
}

func (a *App) commandDefinitions() []CommandDefinition {
	commands := builtinCommands()
	commands = append(commands, a.config.Commands...)
	for i := range commands {
		commands[i].Pinned = slices.Contains(a.config.PinnedItems, commandPinnedID(commands[i].ID))
	}
	return commands
}

func (a *App) findCommand(id string) (CommandDefinition, bool) {
	for _, cmd := range a.commandDefinitions() {
		if cmd.ID == id {
			return cmd, true
		}
	}
	return CommandDefinition{}, false
}

func (a *App) findUserCommand(id string) (CommandDefinition, int, bool) {
	for i, cmd := range a.config.Commands {
		if cmd.ID == id {
			return cmd, i, true
		}
	}
	return CommandDefinition{}, -1, false
}

func (a *App) syncCommandPinnedState(cmd *CommandDefinition) {
	cmd.Pinned = slices.Contains(a.config.PinnedItems, commandPinnedID(cmd.ID))
}

func (a *App) ensurePinnedState(resultID string, pinned bool) {
	canonical := canonicalPinnedID(resultID)
	idx := slices.Index(a.config.PinnedItems, canonical)
	switch {
	case pinned && idx < 0:
		a.config.PinnedItems = append(a.config.PinnedItems, canonical)
	case !pinned && idx >= 0:
		a.config.PinnedItems = append(a.config.PinnedItems[:idx], a.config.PinnedItems[idx+1:]...)
	}
}

func resolveCommandTemplate(template, arg string) string {
	if strings.Contains(template, "{{query}}") {
		return strings.ReplaceAll(template, "{{query}}", url.QueryEscape(arg))
	}
	return template
}

func resolveCommandPreview(cmd CommandDefinition, arg string) string {
	if cmd.RequiresArgument && strings.TrimSpace(arg) == "" {
		return "Type an argument"
	}

	resolved := cmd.Template
	if strings.Contains(cmd.Template, "{{query}}") {
		if cmd.ActionType == "copy_text" || cmd.ActionType == "run_shell" || cmd.ActionType == "open_path" {
			resolved = strings.ReplaceAll(cmd.Template, "{{query}}", arg)
		} else {
			resolved = resolveCommandTemplate(cmd.Template, arg)
		}
	}

	switch cmd.ActionType {
	case "copy_text":
		return resolved
	case "open_path":
		return prettifyPath(filepath.Clean(resolved))
	case "run_shell":
		return resolved
	default:
		return resolved
	}
}

func (a *App) executeCommand(id, arg string) string {
	cmd, ok := a.findCommand(id)
	if !ok {
		return "not found"
	}
	if cmd.RequiresArgument && strings.TrimSpace(arg) == "" {
		return "Type an argument"
	}

	a.usage.Record(commandPinnedID(cmd.ID))
	switch cmd.ActionType {
	case "open_url":
		target := resolveCommandTemplate(cmd.Template, arg)
		runtime.BrowserOpenURL(a.ctx, target)
		runtime.WindowHide(a.ctx)
		a.visible.Store(false)
		return "ok"
	case "copy_text":
		text := cmd.Template
		if strings.Contains(text, "{{query}}") {
			text = strings.ReplaceAll(text, "{{query}}", arg)
		}
		runtime.ClipboardSetText(a.ctx, text)
		return "copied"
	case "open_path":
		target := cmd.Template
		if strings.Contains(target, "{{query}}") {
			target = strings.ReplaceAll(target, "{{query}}", arg)
		}
		shellOpen(target)
		runtime.WindowHide(a.ctx)
		a.visible.Store(false)
		return "ok"
	case "run_shell":
		command := cmd.Template
		if strings.Contains(command, "{{query}}") {
			command = strings.ReplaceAll(command, "{{query}}", arg)
		}
		if err := runCommandTemplate(command, cmd.RunAsAdmin); err != nil {
			return err.Error()
		}
		runtime.WindowHide(a.ctx)
		a.visible.Store(false)
		return "ok"
	}
	return "unknown action"
}

func runCommandTemplate(command string, asAdmin bool) error {
	return startShellCommand(command, asAdmin)
}

func (a *App) GetCommands() []CommandDefinition {
	commands := append([]CommandDefinition(nil), a.config.Commands...)
	for i := range commands {
		a.syncCommandPinnedState(&commands[i])
	}
	return commands
}

func (a *App) SaveCommand(cmd CommandDefinition) error {
	cmd, err := normalizeCommand(cmd)
	if err != nil {
		return err
	}

	if _, idx, ok := a.findUserCommand(cmd.ID); ok {
		a.config.Commands[idx] = cmd
	} else {
		a.config.Commands = append(a.config.Commands, cmd)
	}

	a.ensurePinnedState(commandPinnedID(cmd.ID), cmd.Pinned)
	return a.saveConfig()
}

func (a *App) DeleteCommand(id string) error {
	_, idx, ok := a.findUserCommand(id)
	if !ok {
		return fmt.Errorf("command not found")
	}
	a.config.Commands = append(a.config.Commands[:idx], a.config.Commands[idx+1:]...)
	a.ensurePinnedState(commandPinnedID(id), false)
	return a.saveConfig()
}

func aliasToCommand(trigger, expansion string) CommandDefinition {
	actionType := "copy_text"
	if isURL(expansion) {
		actionType = "open_url"
	}
	title := trigger
	if len(title) > 0 {
		title = strings.ToUpper(title[:1]) + title[1:]
	}
	return CommandDefinition{
		ID:               "legacy-" + trigger,
		Title:            title,
		Keyword:          strings.ToLower(trigger),
		Description:      "Migrated alias",
		ActionType:       actionType,
		Template:         expansion,
		RequiresArgument: false,
	}
}

func duplicateCommand(cmd CommandDefinition) CommandDefinition {
	cmd.ID = "user-" + uuid.NewString()
	cmd.Title = cmd.Title + " Copy"
	cmd.Pinned = false
	return cmd
}

func (app *App) migrateAliasesToCommands() bool {
	if len(app.config.Aliases) == 0 {
		return false
	}

	migrated := false
	for trigger, expansion := range app.config.Aliases {
		commandExists := false
		for _, existingCommand := range app.config.Commands {
			if strings.EqualFold(existingCommand.Keyword, trigger) {
				commandExists = true
				break
			}
		}
		if commandExists {
			continue
		}
		app.config.Commands = append(app.config.Commands, aliasToCommand(trigger, expansion))
		migrated = true
	}

	if migrated {
		app.config.Aliases = nil
	}
	return migrated
}
