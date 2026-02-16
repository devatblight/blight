package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type ClipboardEntry struct {
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}

type ClipboardHistory struct {
	mu      sync.RWMutex
	entries []ClipboardEntry
	path    string
	ctx     context.Context
	maxSize int
}

func NewClipboardHistory(ctx context.Context) *ClipboardHistory {
	home, _ := os.UserHomeDir()
	h := &ClipboardHistory{
		path:    filepath.Join(home, ".blight", "clipboard.json"),
		ctx:     ctx,
		maxSize: 50,
	}
	h.load()
	return h
}

func (h *ClipboardHistory) Add(content string) {
	if content == "" {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.entries) > 0 && h.entries[0].Content == content {
		return
	}

	entry := ClipboardEntry{
		Content:   content,
		Timestamp: time.Now().Unix(),
	}

	h.entries = append([]ClipboardEntry{entry}, h.entries...)

	if len(h.entries) > h.maxSize {
		h.entries = h.entries[:h.maxSize]
	}

	go h.save()
}

func (h *ClipboardHistory) Entries() []ClipboardEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()
	result := make([]ClipboardEntry, len(h.entries))
	copy(result, h.entries)
	return result
}

func (h *ClipboardHistory) CopyToClipboard(index int) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if index < 0 || index >= len(h.entries) {
		return false
	}

	runtime.ClipboardSetText(h.ctx, h.entries[index].Content)
	return true
}

func (h *ClipboardHistory) PollClipboard() {
	lastContent := ""
	for {
		text, err := runtime.ClipboardGetText(h.ctx)
		if err == nil && text != lastContent && text != "" {
			lastContent = text
			h.Add(text)
		}
		time.Sleep(1 * time.Second)
	}
}

func (h *ClipboardHistory) load() {
	data, err := os.ReadFile(h.path)
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	json.Unmarshal(data, &h.entries)
}

func (h *ClipboardHistory) save() {
	h.mu.RLock()
	data, _ := json.MarshalIndent(h.entries, "", "  ")
	h.mu.RUnlock()
	os.MkdirAll(filepath.Dir(h.path), 0755)
	os.WriteFile(h.path, data, 0644)
}
