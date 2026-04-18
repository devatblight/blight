package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"blight/internal/commands"
	"blight/internal/debug"
	"blight/internal/search"
)

var builtinCommands = []CommandDefinition{
	{ID: "g", Title: "Google Search", Keyword: "g", Description: "Search Google", ActionType: "open_url", Template: "https://www.google.com/search?q={{query}}", RequiresArgument: true},
	{ID: "gh", Title: "GitHub Search", Keyword: "gh", Description: "Search GitHub", ActionType: "open_url", Template: "https://github.com/search?q={{query}}", RequiresArgument: true},
	{ID: "yt", Title: "YouTube", Keyword: "yt", Description: "Search YouTube", ActionType: "open_url", Template: "https://www.youtube.com/results?search_query={{query}}", RequiresArgument: true},
	{ID: "wiki", Title: "Wikipedia", Keyword: "wiki", Description: "Search Wikipedia", ActionType: "open_url", Template: "https://en.wikipedia.org/wiki/Special:Search?search={{query}}", RequiresArgument: true},
	{ID: "maps", Title: "Google Maps", Keyword: "maps", Description: "Search Maps", ActionType: "open_url", Template: "https://maps.google.com/?q={{query}}", RequiresArgument: true},
}

func (a *App) maxResults() int {
	if a.config.MaxResults > 0 {
		return a.config.MaxResults
	}
	return 8
}

// allCommands returns built-ins merged with user commands. User commands override built-ins by keyword.
func (a *App) allCommands() []CommandDefinition {
	userKeywords := make(map[string]bool)
	for _, c := range a.config.Commands {
		userKeywords[strings.ToLower(c.Keyword)] = true
	}
	var result []CommandDefinition
	for _, b := range builtinCommands {
		if !userKeywords[strings.ToLower(b.Keyword)] {
			result = append(result, b)
		}
	}
	return append(result, a.config.Commands...)
}

// resolveCommandTemplate substitutes {{query}} in the template.
func resolveCommandTemplate(cmd CommandDefinition, arg string) string {
	if cmd.ActionType == "open_url" {
		return strings.ReplaceAll(cmd.Template, "{{query}}", url.QueryEscape(arg))
	}
	return strings.ReplaceAll(cmd.Template, "{{query}}", arg)
}

func commandResultID(actionType, resolved string) string {
	switch actionType {
	case "open_url":
		return "cmd-url:" + resolved
	case "copy_text":
		return "cmd-copy:" + resolved
	case "open_path":
		return "cmd-path:" + resolved
	case "run_shell":
		return "cmd-shell:" + resolved
	default:
		return "cmd-url:" + resolved
	}
}

// searchCommands returns commands matching term. Empty term returns all commands.
func (a *App) searchCommands(term string) []SearchResult {
	var results []SearchResult
	termLower := strings.ToLower(strings.TrimSpace(term))
	for _, cmd := range a.allCommands() {
		kw := strings.ToLower(cmd.Keyword)
		title := strings.ToLower(cmd.Title)
		desc := strings.ToLower(cmd.Description)
		if termLower == "" ||
			strings.HasPrefix(kw, termLower) ||
			strings.Contains(title, termLower) ||
			strings.Contains(desc, termLower) ||
			strings.HasPrefix(termLower, kw+" ") ||
			termLower == kw {
			arg := ""
			if strings.HasPrefix(termLower, kw+" ") {
				arg = term[len(kw)+1:]
			}
			var id, subtitle string
			if cmd.RequiresArgument && arg == "" {
				id = "cmd-needs-arg:" + cmd.ID
				subtitle = cmd.Description
				if subtitle == "" {
					subtitle = "Type an argument"
				}
			} else {
				resolved := resolveCommandTemplate(cmd, arg)
				id = commandResultID(cmd.ActionType, resolved)
				subtitle = resolved
			}
			results = append(results, SearchResult{
				ID:       id,
				Title:    cmd.Title,
				Subtitle: subtitle,
				Icon:     cmd.Icon,
				Category: "Commands",
			})
		}
	}
	if len(results) == 0 {
		return enrichResults([]SearchResult{{
			ID:       "web-search:" + term,
			Title:    "Search the web for \"" + term + "\"",
			Subtitle: "Opens in your default browser",
			Category: "Web",
		}})
	}
	return enrichResults(results)
}

func (a *App) Search(query string) []SearchResult {
	log := debug.Get()
	if query == "" {
		return a.getDefaultResults()
	}

	if strings.HasPrefix(query, ">") {
		return a.searchCommands(strings.TrimPrefix(query, ">"))
	}

	if strings.HasPrefix(query, "~") || isAbsPath(query) {
		return a.searchPath(query)
	}

	caps := search.DefaultCaps()
	var scored []search.Scored[SearchResult]

	if isURL(query) {
		scored = append(scored, search.Scored[SearchResult]{
			Item:  SearchResult{ID: "url-open:" + query, Title: "Open URL", Subtitle: query, Category: "Web"},
			Score: 9500,
			Cat:   "Web",
		})
	}

	if len(a.config.Aliases) > 0 {
		qLower := strings.ToLower(query)
		for trigger, expansion := range a.config.Aliases {
			if strings.HasPrefix(strings.ToLower(trigger), qLower) {
				scored = append(scored, search.Scored[SearchResult]{
					Item:  SearchResult{ID: "alias:" + trigger, Title: trigger, Subtitle: expansion, Category: "Aliases"},
					Score: 4000,
					Cat:   "Aliases",
				})
			}
		}
	}

	scored = append(scored, a.searchCommandsScored(query)...)

	if commands.IsCalcQuery(query) {
		calc := commands.Evaluate(query)
		if calc.Valid {
			scored = append(scored, search.Scored[SearchResult]{
				Item:  SearchResult{ID: "calc-result", Title: calc.Result, Subtitle: calc.Expression + " — press Enter to copy", Category: "Calculator"},
				Score: 9000,
				Cat:   "Calculator",
			})
		}
	}

	queryLower := strings.ToLower(query)
	if strings.HasPrefix(queryLower, "cb ") || strings.HasPrefix(queryLower, "clip ") || queryLower == "clipboard" || queryLower == "cb" || queryLower == "clip" {
		for i, entry := range a.clipboard.Entries() {
			if i >= caps["Clipboard"] {
				break
			}
			preview := entry.Content
			if len(preview) > 80 {
				preview = preview[:80] + "…"
			}
			scored = append(scored, search.Scored[SearchResult]{
				Item:  SearchResult{ID: fmt.Sprintf("clip-%d", i), Title: preview, Subtitle: "Clipboard — press Enter to copy", Category: "Clipboard"},
				Score: 8000 - i*10,
				Cat:   "Clipboard",
			})
		}
	}

	scored = append(scored, a.searchSystemCommandsScored(query)...)
	scored = append(scored, a.searchAppsScored(query)...)
	scored = append(scored, a.searchDirsScored(query)...)
	scored = append(scored, a.searchFilesScored(query)...)

	results := search.RankAndCap(scored, caps)

	if len(results) == 0 {
		log.Debug("search no results — web fallback", map[string]interface{}{"query": query})
		return enrichResults([]SearchResult{{
			ID:       "web-search:" + query,
			Title:    "Search the web for \"" + query + "\"",
			Subtitle: "Opens in your default browser",
			Category: "Web",
		}})
	}

	results = append(results, SearchResult{
		ID:       "web-search:" + query,
		Title:    "Search the web for \"" + query + "\"",
		Subtitle: "Opens in your default browser",
		Category: "Web",
	})

	log.Debug("search", map[string]interface{}{"query": query, "results": len(results)})
	return enrichResults(results)
}

func (a *App) searchCommandsScored(query string) []search.Scored[SearchResult] {
	qLower := strings.ToLower(query)
	var out []search.Scored[SearchResult]
	for _, cmd := range a.allCommands() {
		kw := strings.ToLower(cmd.Keyword)
		kwMatch := qLower == kw || strings.HasPrefix(qLower, kw+" ")
		kwPrefix := strings.HasPrefix(kw, qLower) && qLower != kw
		if !kwMatch && !kwPrefix {
			continue
		}
		arg := ""
		if strings.HasPrefix(qLower, kw+" ") {
			arg = query[len(kw)+1:]
		}
		var id, subtitle string
		if cmd.RequiresArgument && arg == "" {
			id = "cmd-needs-arg:" + cmd.ID
			subtitle = "Type an argument"
		} else {
			resolved := resolveCommandTemplate(cmd, arg)
			id = commandResultID(cmd.ActionType, resolved)
			subtitle = resolved
		}
		s := 3000
		if kwMatch {
			s = 8000
		}
		if cmd.Pinned {
			s += 3000
		}
		if id != "cmd-needs-arg:"+cmd.ID {
			s += a.usage.Score(id)
		}
		out = append(out, search.Scored[SearchResult]{
			Item:  SearchResult{ID: id, Title: cmd.Title, Subtitle: subtitle, Icon: cmd.Icon, Category: "Commands"},
			Score: s,
			Cat:   "Commands",
		})
	}
	return out
}

func (a *App) searchSystemCommandsScored(query string) []search.Scored[SearchResult] {
	queryLower := strings.ToLower(query)
	var out []search.Scored[SearchResult]
	for _, cmd := range commands.SystemCommands {
		cmdName := strings.ToLower(cmd.Name)
		matched := false
		s := 0
		if cmdName == queryLower {
			matched = true
			s = 10000
		} else if strings.HasPrefix(cmdName, queryLower) {
			matched = true
			s = 5000 + len(queryLower)*10
		} else if strings.Contains(cmdName, queryLower) {
			matched = true
			s = 2000 + len(queryLower)*5
		}
		if !matched {
			for _, keyword := range cmd.Keywords {
				if strings.Contains(keyword, queryLower) {
					matched = true
					s = 1500
					break
				}
			}
		}
		if !matched {
			continue
		}
		s += a.usage.Score("sys-" + cmd.ID)
		out = append(out, search.Scored[SearchResult]{
			Item:  SearchResult{ID: "sys-" + cmd.ID, Title: cmd.Name, Subtitle: cmd.Subtitle, Icon: cmd.Icon, Category: "System"},
			Score: s,
			Cat:   "System",
		})
	}
	return out
}

func (a *App) searchAppsScored(query string) []search.Scored[SearchResult] {
	allApps := a.scanner.Apps()
	names := a.scanner.Names()
	usageScores := make([]int, len(allApps))
	for i, app := range allApps {
		usageScores[i] = a.usage.Score(app.Name)
		if slices.Contains(a.config.PinnedItems, app.Name) {
			usageScores[i] += 100
		}
	}
	matches := search.Fuzzy(query, names, usageScores)
	limit := min(len(matches), a.maxResults())
	out := make([]search.Scored[SearchResult], 0, limit)
	for _, m := range matches[:limit] {
		app := allApps[m.Index]
		subtitle := "Application"
		if !app.IsLnk {
			subtitle = prettifyPath(app.Path)
		}
		out = append(out, search.Scored[SearchResult]{
			Item:  SearchResult{ID: app.Name, Title: app.Name, Subtitle: subtitle, Category: "Applications", Path: app.Path},
			Score: m.Score,
			Cat:   "Applications",
		})
	}
	return out
}

func (a *App) searchDirsScored(query string) []search.Scored[SearchResult] {
	if len(query) < 2 || a.config.DisableFolderIndex {
		return nil
	}
	status := a.fileIdx.Status()
	if status.State != "ready" {
		return nil
	}
	allScores := a.usage.AllScores()
	dirScores := make(map[string]int, len(allScores))
	for k, v := range allScores {
		if strings.HasPrefix(k, "dir-open:") {
			dirScores[strings.TrimPrefix(k, "dir-open:")] = v
		}
	}
	dirResults := a.fileIdx.SearchDirs(query, dirScores)
	limit := min(len(dirResults), max(3, a.maxResults()/2))
	out := make([]search.Scored[SearchResult], 0, limit)
	for i, d := range dirResults[:limit] {
		s := 5000 - i*50
		if us := dirScores[d.Path]; us > 0 {
			s += us
		}
		out = append(out, search.Scored[SearchResult]{
			Item:  SearchResult{ID: "dir-open:" + d.Path, Title: d.Name, Subtitle: prettifyPath(d.Path), Category: "Folders", Path: d.Path},
			Score: s,
			Cat:   "Folders",
		})
	}
	return out
}

func (a *App) searchFilesScored(query string) []search.Scored[SearchResult] {
	if len(query) < 2 {
		return nil
	}
	status := a.fileIdx.Status()
	if status.State != "ready" {
		return nil
	}
	allScores := a.usage.AllScores()
	fileScores := make(map[string]int, len(allScores))
	for k, v := range allScores {
		if strings.HasPrefix(k, "file-open:") {
			fileScores[strings.TrimPrefix(k, "file-open:")] = v
		}
	}
	fileResults := a.fileIdx.SearchFiles(query, fileScores)
	limit := min(len(fileResults), a.maxResults())
	out := make([]search.Scored[SearchResult], 0, limit)
	for i, f := range fileResults[:limit] {
		s := 4000 - i*50
		if us := fileScores[f.Path]; us > 0 {
			s += us
		}
		out = append(out, search.Scored[SearchResult]{
			Item:  SearchResult{ID: "file-open:" + f.Path, Title: f.Name, Subtitle: prettifyPath(f.Dir), Category: "Files", Path: f.Path},
			Score: s,
			Cat:   "Files",
		})
	}
	return out
}

func (a *App) getDefaultResults() []SearchResult {
	allApps := a.scanner.Apps()
	var results []SearchResult

	pinnedSet := make(map[string]bool)
	for _, pinnedID := range a.config.PinnedItems {
		pinnedSet[pinnedID] = true
		for _, app := range allApps {
			if app.Name == pinnedID {
				subtitle := "Application"
				if !app.IsLnk {
					subtitle = prettifyPath(app.Path)
				}
				results = append(results, SearchResult{
					ID:       app.Name,
					Title:    app.Name,
					Subtitle: subtitle,
					Category: "Pinned",
					Path:     app.Path,
				})
				break
			}
		}
	}

	names := a.scanner.Names()
	usageScores := make([]int, len(allApps))
	for i, app := range allApps {
		usageScores[i] = a.usage.Score(app.Name)
	}
	matches := search.Fuzzy("", names, usageScores)

	count := 6
	added := 0
	for _, match := range matches {
		if added >= count {
			break
		}
		app := allApps[match.Index]
		if pinnedSet[app.Name] {
			continue
		}
		category := "Suggested"
		if a.usage.Score(app.Name) > 0 {
			category = "Recent"
		}
		subtitle := "Application"
		if !app.IsLnk {
			subtitle = prettifyPath(app.Path)
		}
		results = append(results, SearchResult{
			ID:       app.Name,
			Title:    app.Name,
			Subtitle: subtitle,
			Category: category,
			Path:     app.Path,
		})
		added++
	}

	if a.clipboard != nil {
		for i, entry := range a.clipboard.Entries() {
			if i >= 3 {
				break
			}
			title := entry.Content
			if len(title) > 60 {
				title = title[:60] + "…"
			}
			results = append(results, SearchResult{
				ID:       fmt.Sprintf("clip-%d", i),
				Title:    title,
				Subtitle: "Clipboard",
				Category: "Clipboard",
			})
		}
	}

	return enrichResults(results)
}

func (a *App) searchPath(query string) []SearchResult {
	home, _ := os.UserHomeDir()

	expanded := filepath.FromSlash(query)
	if strings.HasPrefix(expanded, "~") {
		expanded = home + expanded[1:]
	}

	var searchDir, filter string
	last := expanded[len(expanded)-1]
	if last == filepath.Separator || last == '/' {
		searchDir = filepath.Clean(expanded)
		filter = ""
	} else {
		searchDir = filepath.Dir(expanded)
		filter = filepath.Base(expanded)
		if searchDir == "." {
			searchDir = home
		}
	}

	entries, err := os.ReadDir(searchDir)
	if err != nil {
		return []SearchResult{{
			ID:       "no-results",
			Title:    "Directory not found",
			Subtitle: prettifyPath(searchDir),
			Category: "Files",
		}}
	}

	filterLower := strings.ToLower(filter)
	limit := a.maxResults() * 2

	var dirs, fileResults []SearchResult
	for _, entry := range entries {
		if len(dirs)+len(fileResults) >= limit {
			break
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if filterLower != "" && !strings.Contains(strings.ToLower(name), filterLower) {
			continue
		}
		path := filepath.Join(searchDir, name)
		if entry.IsDir() {
			dirs = append(dirs, SearchResult{
				ID:       "dir-open:" + path,
				Title:    name,
				Subtitle: prettifyPath(path),
				Category: "Folders",
				Path:     path,
			})
		} else {
			fileResults = append(fileResults, SearchResult{
				ID:       "file-open:" + path,
				Title:    name,
				Subtitle: prettifyPath(searchDir),
				Category: "Files",
				Path:     path,
			})
		}
	}

	results := append(dirs, fileResults...)
	if len(results) == 0 {
		return []SearchResult{{
			ID:       "no-results",
			Title:    "No matches in " + prettifyPath(searchDir),
			Subtitle: "Try a different name",
			Category: "Files",
		}}
	}
	return enrichResults(results)
}
