package apps

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type AppEntry struct {
	Name    string
	Path    string
	LnkPath string
	IsLnk   bool
}

type Scanner struct {
	mu   sync.RWMutex
	apps []AppEntry
}

func NewScanner() *Scanner {
	s := &Scanner{}
	s.Scan()
	return s
}

func (s *Scanner) Scan() {
	var found []AppEntry

	for _, dir := range s.startMenuDirs() {
		found = append(found, scanDir(dir)...)
	}

	found = append(found, scanPathApps()...)
	found = deduplicate(found)

	s.mu.Lock()
	s.apps = found
	s.mu.Unlock()
}

func (s *Scanner) Apps() []AppEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]AppEntry, len(s.apps))
	copy(result, s.apps)
	return result
}

func (s *Scanner) Names() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	names := make([]string, len(s.apps))
	for i, app := range s.apps {
		names[i] = app.Name
	}
	return names
}

func (s *Scanner) startMenuDirs() []string {
	var dirs []string
	if d := os.Getenv("ProgramData"); d != "" {
		dirs = append(dirs, filepath.Join(d, "Microsoft", "Windows", "Start Menu", "Programs"))
	}
	if d := os.Getenv("AppData"); d != "" {
		dirs = append(dirs, filepath.Join(d, "Microsoft", "Windows", "Start Menu", "Programs"))
	}
	return dirs
}

var ignoredPathSegments = []string{
	"system32", "syswow64", "winsxs", "servicing",
	"windows defender", "windows nt", "windows mail",
	"windows sidebar", "windows media player",
	"maintenance", "accessibility",
}

var ignoredNamePrefixes = []string{
	"uninstall", "readme", "help", "license", "changelog",
	"release notes", "setup", "install", "update",
}

func scanDir(root string) []AppEntry {
	var results []AppEntry

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".lnk" && ext != ".exe" {
			return nil
		}

		if isIgnoredPath(path) {
			return nil
		}

		name := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
		if isIgnoredName(name) {
			return nil
		}

		results = append(results, AppEntry{
			Name:    name,
			Path:    path,
			LnkPath: path,
			IsLnk:   ext == ".lnk",
		})
		return nil
	})

	return results
}

func scanPathApps() []AppEntry {
	var results []AppEntry
	seen := make(map[string]bool)

	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if isIgnoredPath(dir) {
			continue
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if strings.ToLower(filepath.Ext(entry.Name())) != ".exe" {
				continue
			}

			name := strings.TrimSuffix(entry.Name(), ".exe")
			lower := strings.ToLower(name)

			if seen[lower] || isIgnoredName(name) {
				continue
			}
			seen[lower] = true

			results = append(results, AppEntry{
				Name: name,
				Path: filepath.Join(dir, entry.Name()),
			})
		}
	}

	return results
}

func isIgnoredPath(path string) bool {
	lower := strings.ToLower(path)
	for _, seg := range ignoredPathSegments {
		if strings.Contains(lower, seg) {
			return true
		}
	}
	parts := strings.Split(path, string(filepath.Separator))
	for _, part := range parts {
		if strings.HasPrefix(part, ".") && len(part) > 1 {
			return true
		}
	}
	return false
}

func isIgnoredName(name string) bool {
	lower := strings.ToLower(name)
	for _, prefix := range ignoredNamePrefixes {
		if strings.Contains(lower, prefix) {
			return true
		}
	}
	return false
}

func deduplicate(apps []AppEntry) []AppEntry {
	seen := make(map[string]bool)
	var result []AppEntry
	for _, app := range apps {
		key := strings.ToLower(app.Name)
		if !seen[key] {
			seen[key] = true
			result = append(result, app)
		}
	}
	return result
}
