package search

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// usageEntry stores launch count and the last time the item was used.
type usageEntry struct {
	Count    int   `json:"count"`
	LastUsed int64 `json:"lastUsed"` // Unix timestamp
}

type UsageTracker struct {
	mu      sync.RWMutex
	entries map[string]usageEntry
	path    string
}

func NewUsageTracker() *UsageTracker {
	home, _ := os.UserHomeDir()
	tracker := &UsageTracker{
		entries: make(map[string]usageEntry),
		path:    filepath.Join(home, ".blight", "usage.json"),
	}
	tracker.load()
	return tracker
}

// Record increments the usage count for id and updates the last-used timestamp.
func (t *UsageTracker) Record(id string) {
	t.mu.Lock()
	entry := t.entries[id]
	entry.Count++
	entry.LastUsed = time.Now().Unix()
	t.entries[id] = entry
	t.mu.Unlock()
	go t.save()
}

// Score returns a decayed usage score for id.
// Recent uses contribute more than old ones.
// Score = count * recencyFactor, where recencyFactor ∈ (0, 1] decays over ~30 days.
func (t *UsageTracker) Score(id string) int {
	t.mu.RLock()
	entry, ok := t.entries[id]
	t.mu.RUnlock()

	if !ok || entry.Count == 0 {
		return 0
	}

	daysSince := time.Since(time.Unix(entry.LastUsed, 0)).Hours() / 24
	// Decay half-life ≈ 30 days: factor = 0.5^(days/30)
	recency := math.Pow(0.5, daysSince/30.0)
	return int(float64(entry.Count)*recency*100) + 1
}

func (t *UsageTracker) load() {
	data, err := os.ReadFile(t.path)
	if err != nil {
		return
	}

	// Support both old format (map[string]int) and new format (map[string]usageEntry)
	var newFmt map[string]usageEntry
	if err := json.Unmarshal(data, &newFmt); err == nil {
		t.entries = newFmt
		return
	}

	// Try migrating from old counts-only format
	var oldFmt map[string]int
	if err := json.Unmarshal(data, &oldFmt); err == nil {
		for id, count := range oldFmt {
			t.entries[id] = usageEntry{Count: count, LastUsed: time.Now().Unix()}
		}
		go t.save() // persist migrated format
	}
}

func (t *UsageTracker) save() {
	t.mu.RLock()
	data, err := json.MarshalIndent(t.entries, "", "  ")
	t.mu.RUnlock()
	if err != nil {
		return
	}
	os.MkdirAll(filepath.Dir(t.path), 0755)
	os.WriteFile(t.path, data, 0644)
}
