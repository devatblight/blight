package files

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type FileEntry struct {
	Name string
	Path string
	Dir  string
	Ext  string
	Size int64
}

type IndexStatus struct {
	State   string `json:"state"`
	Message string `json:"message"`
	Count   int    `json:"count"`
	Total   int    `json:"total"`
}

type FileIndex struct {
	mu       sync.RWMutex
	files    []FileEntry
	names    []string
	status   atomic.Value
	onStatus func(IndexStatus)
}

func NewFileIndex(onStatus func(IndexStatus)) *FileIndex {
	idx := &FileIndex{
		onStatus: onStatus,
	}
	idx.status.Store(IndexStatus{State: "idle", Message: "Not indexed"})
	return idx
}

func (idx *FileIndex) Start() {
	go idx.buildIndex()
}

func (idx *FileIndex) Reindex() {
	go idx.buildIndex()
}

func (idx *FileIndex) ClearIndex() {
	idx.mu.Lock()
	idx.files = nil
	idx.names = nil
	idx.mu.Unlock()
	idx.setStatus(IndexStatus{State: "idle", Message: "Index cleared"})
}

func (idx *FileIndex) Status() IndexStatus {
	return idx.status.Load().(IndexStatus)
}

func (idx *FileIndex) Files() []FileEntry {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	result := make([]FileEntry, len(idx.files))
	copy(result, idx.files)
	return result
}

func (idx *FileIndex) Names() []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	result := make([]string, len(idx.names))
	copy(result, idx.names)
	return result
}

func (idx *FileIndex) setStatus(s IndexStatus) {
	idx.status.Store(s)
	if idx.onStatus != nil {
		idx.onStatus(s)
	}
}

func (idx *FileIndex) buildIndex() {
	idx.setStatus(IndexStatus{State: "indexing", Message: "Scanning files..."})
	idx.manualIndex()
}

func (idx *FileIndex) manualIndex() {
	home, _ := os.UserHomeDir()
	scanDirs := []string{
		filepath.Join(home, "Desktop"),
		filepath.Join(home, "Documents"),
		filepath.Join(home, "Downloads"),
		filepath.Join(home, "Pictures"),
		filepath.Join(home, "Videos"),
		filepath.Join(home, "Music"),
	}

	if projects := filepath.Join(home, "Projects"); dirExists(projects) {
		scanDirs = append(scanDirs, projects)
	}
	if code := filepath.Join(home, "code"); dirExists(code) {
		scanDirs = append(scanDirs, code)
	}

	var allFiles []FileEntry
	count := 0
	total := 0

	for _, dir := range scanDirs {
		total += estimateCount(dir)
	}
	if total == 0 {
		total = 1
	}

	start := time.Now()
	lastUpdate := time.Now()

	for _, dir := range scanDirs {
		dirName := filepath.Base(dir)
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			name := info.Name()

			if info.IsDir() {
				if shouldSkipDir(name) {
					return filepath.SkipDir
				}
				return nil
			}

			count++
			// Throttle status updates to every 200ms to avoid UI spam
			if time.Since(lastUpdate) > 200*time.Millisecond {
				lastUpdate = time.Now()
				idx.setStatus(IndexStatus{
					State:   "indexing",
					Message: "Scanning " + dirName + "...",
					Count:   count,
					Total:   total,
				})
			}

			allFiles = append(allFiles, FileEntry{
				Name: name,
				Path: path,
				Dir:  filepath.Dir(path),
				Ext:  strings.ToLower(filepath.Ext(name)),
				Size: info.Size(),
			})

			return nil
		})
	}

	names := make([]string, len(allFiles))
	for i, f := range allFiles {
		names[i] = f.Name
	}

	idx.mu.Lock()
	idx.files = allFiles
	idx.names = names
	idx.mu.Unlock()

	elapsed := time.Since(start).Round(time.Millisecond)
	msg := fmt.Sprintf("%d files indexed in %s", count, elapsed)
	idx.setStatus(IndexStatus{
		State:   "ready",
		Message: msg,
		Count:   count,
		Total:   count,
	})
}

// SearchFiles performs substring matching against the local index.
// Matches on filename, extension, and path components.
func (idx *FileIndex) SearchFiles(query string) []FileEntry {
	if query == "" {
		return nil
	}

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	q := strings.ToLower(query)
	var results []FileEntry

	for _, f := range idx.files {
		nameLower := strings.ToLower(f.Name)
		pathLower := strings.ToLower(f.Path)
		if strings.Contains(nameLower, q) || strings.Contains(pathLower, q) {
			results = append(results, f)
			if len(results) >= 15 {
				break
			}
		}
	}

	return results
}

func shouldSkipDir(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	skip := map[string]bool{
		"node_modules": true, "__pycache__": true, "vendor": true,
		"$RECYCLE.BIN": true, "System Volume Information": true,
		"AppData": true, "cache": true, "Cache": true,
		"dist": true, "build": true, "target": true,
		"venv": true, ".venv": true, "env": true,
	}
	return skip[name]
}

func estimateCount(dir string) int {
	count := 0
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	for _, entry := range entries {
		if entry.IsDir() {
			count += 500
		} else {
			count++
		}
	}
	return count
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// hiddenCmd sets up a command to run with a hidden window on Windows.
func HiddenCmd(name string, args ...string) *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		HideWindow: true,
	}
}
