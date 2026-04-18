package main

import (
	"os"
	"strings"
	goruntime "runtime"
)

// enrichResult sets Kind, PrimaryActionLabel, SecondaryActionLabel, and
// SupportsActions on a SearchResult based on its Category and ID.
func enrichResult(r SearchResult) SearchResult {
	switch r.Category {
	case "Applications", "Pinned", "Recent", "Suggested":
		r.Kind = "app"
		r.PrimaryActionLabel = "Open"
		r.SecondaryActionLabel = "Run as admin"
		r.SupportsActions = true
	case "Commands", "Aliases":
		r.Kind = "command"
		r.PrimaryActionLabel = "Run"
		if !strings.HasPrefix(r.ID, "cmd-needs-arg:") {
			r.SecondaryActionLabel = "Copy value"
			r.SupportsActions = true
		}
	case "Files":
		if r.ID == "no-results" {
			return r
		}
		r.Kind = "file"
		r.PrimaryActionLabel = "Open"
		r.SecondaryActionLabel = "Show in Explorer"
		r.SupportsActions = true
	case "Folders":
		r.Kind = "folder"
		r.PrimaryActionLabel = "Open"
		r.SecondaryActionLabel = "Open in Terminal"
		r.SupportsActions = true
	case "Clipboard":
		r.Kind = "clipboard"
		r.PrimaryActionLabel = "Copy"
		r.SupportsActions = true
	case "System":
		r.Kind = "system"
		r.PrimaryActionLabel = "Run"
	case "Web":
		r.Kind = "web"
		r.PrimaryActionLabel = "Search"
	case "Calculator":
		r.Kind = "calc"
		r.PrimaryActionLabel = "Copy result"
	}
	return r
}

func enrichResults(rs []SearchResult) []SearchResult {
	for i := range rs {
		rs[i] = enrichResult(rs[i])
	}
	return rs
}

// icon returns a Segoe MDL2/Fluent glyph on Windows and a plain emoji on other platforms.
// Segoe PUA codepoints are meaningless outside Windows, so we fall back to emoji elsewhere.
func icon(winGlyph, fallback string) string {
	if goruntime.GOOS == "windows" {
		return winGlyph
	}
	return fallback
}

// revealLabel returns the platform-appropriate label for the "show in file manager" action.
func revealLabel() string {
	switch goruntime.GOOS {
	case "windows":
		return "Show in Explorer"
	case "darwin":
		return "Reveal in Finder"
	default:
		return "Show in Files"
	}
}

// elevateLabel returns the platform-appropriate label for the "run with elevated privileges" action.
func elevateLabel() string {
	if goruntime.GOOS == "windows" {
		return "Run as Administrator"
	}
	return "Run as Root"
}

// prettifyPath replaces the home directory prefix with ~.
func prettifyPath(path string) string {
	home, _ := os.UserHomeDir()
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

// isURL returns true if s looks like an http/https/ftp URL.
func isURL(s string) bool {
	sl := strings.ToLower(s)
	return strings.HasPrefix(sl, "http://") ||
		strings.HasPrefix(sl, "https://") ||
		strings.HasPrefix(sl, "ftp://")
}

// isAbsPath returns true if s looks like an absolute filesystem path on any
// supported platform: Windows drive paths (C:\, C:/), UNC paths (\\server\),
// and Unix-style absolute paths (/home/...).
func isAbsPath(s string) bool {
	if len(s) >= 2 && s[1] == ':' {
		return true // Windows drive: C:\ or C:/
	}
	if strings.HasPrefix(s, `\\`) || strings.HasPrefix(s, "//") {
		return true // UNC / network path
	}
	if strings.HasPrefix(s, "/") {
		return true // Unix absolute path
	}
	return false
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
