package search

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type UsageTracker struct {
	mu     sync.RWMutex
	counts map[string]int
	path   string
}

func NewUsageTracker() *UsageTracker {
	home, _ := os.UserHomeDir()
	t := &UsageTracker{
		counts: make(map[string]int),
		path:   filepath.Join(home, ".blight", "usage.json"),
	}
	t.load()
	return t
}

func (t *UsageTracker) Record(id string) {
	t.mu.Lock()
	t.counts[id]++
	t.mu.Unlock()
	go t.save()
}

func (t *UsageTracker) Score(id string) int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.counts[id]
}

func (t *UsageTracker) load() {
	data, err := os.ReadFile(t.path)
	if err != nil {
		return
	}
	json.Unmarshal(data, &t.counts)
}

func (t *UsageTracker) save() {
	t.mu.RLock()
	data, err := json.MarshalIndent(t.counts, "", "  ")
	t.mu.RUnlock()
	if err != nil {
		return
	}
	os.MkdirAll(filepath.Dir(t.path), 0755)
	os.WriteFile(t.path, data, 0644)
}
