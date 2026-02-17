package debug

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strings"
)

const consoleHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>blight — debug console</title>
<style>
  @import url('https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;600&display=swap');

  * { margin: 0; padding: 0; box-sizing: border-box; }

  body {
    background: #0d0d0d;
    color: #d4d4d4;
    font-family: 'JetBrains Mono', 'Consolas', monospace;
    font-size: 12px;
    line-height: 1.6;
    display: flex;
    flex-direction: column;
    height: 100vh;
  }

  #header {
    background: #161616;
    border-bottom: 1px solid #2a2a2a;
    padding: 10px 16px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-shrink: 0;
  }

  #header h1 {
    font-size: 13px;
    font-weight: 600;
    color: #e0e0e0;
    display: flex;
    align-items: center;
    gap: 8px;
  }

  #header h1 .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #4ade80;
    animation: pulse 2s infinite;
  }

  #header h1 .dot.disconnected {
    background: #f87171;
    animation: none;
  }

  @keyframes pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.4; }
  }

  .header-actions {
    display: flex;
    gap: 8px;
  }

  .header-actions button {
    background: #252525;
    border: 1px solid #333;
    color: #999;
    padding: 4px 10px;
    border-radius: 4px;
    font-family: inherit;
    font-size: 11px;
    cursor: pointer;
    transition: all 0.15s;
  }

  .header-actions button:hover {
    background: #333;
    color: #e0e0e0;
  }

  #filter-bar {
    background: #131313;
    border-bottom: 1px solid #222;
    padding: 6px 16px;
    display: flex;
    align-items: center;
    gap: 8px;
    flex-shrink: 0;
  }

  .filter-btn {
    background: none;
    border: 1px solid transparent;
    color: #666;
    padding: 3px 8px;
    border-radius: 3px;
    font-family: inherit;
    font-size: 11px;
    cursor: pointer;
    transition: all 0.15s;
  }

  .filter-btn:hover { color: #aaa; }
  .filter-btn.active { color: #e0e0e0; border-color: #444; background: #1e1e1e; }
  .filter-btn.level-error.active { color: #f87171; border-color: #7f1d1d; }
  .filter-btn.level-warn.active { color: #fbbf24; border-color: #78350f; }
  .filter-btn.level-info.active { color: #60a5fa; border-color: #1e3a5f; }
  .filter-btn.level-debug.active { color: #a78bfa; border-color: #4c1d95; }

  #search-input {
    background: #1a1a1a;
    border: 1px solid #333;
    color: #d4d4d4;
    padding: 3px 8px;
    border-radius: 3px;
    font-family: inherit;
    font-size: 11px;
    margin-left: auto;
    width: 200px;
    outline: none;
  }

  #search-input:focus { border-color: #555; }
  #search-input::placeholder { color: #555; }

  #log-container {
    flex: 1;
    overflow-y: auto;
    padding: 4px 0;
  }

  #log-container::-webkit-scrollbar { width: 6px; }
  #log-container::-webkit-scrollbar-track { background: #0d0d0d; }
  #log-container::-webkit-scrollbar-thumb { background: #333; border-radius: 3px; }

  .log-entry {
    padding: 3px 16px;
    display: flex;
    flex-wrap: wrap;
    align-items: flex-start;
    gap: 4px 10px;
    border-bottom: 1px solid #1a1a1a;
    transition: background 0.1s;
    cursor: pointer;
    position: relative;
  }

  .log-entry:hover { background: #151515; }
  .log-entry.level-error { background: rgba(248, 113, 113, 0.04); }
  .log-entry.level-error:hover { background: rgba(248, 113, 113, 0.08); }
  .log-entry.level-warn { background: rgba(251, 191, 36, 0.03); }
  .log-entry.level-fatal { background: rgba(248, 113, 113, 0.1); border-left: 3px solid #f87171; }

  .log-summary {
    display: flex;
    align-items: flex-start;
    gap: 10px;
    width: 100%;
    min-width: 0;
  }

  .log-chevron {
    flex-shrink: 0;
    color: #444;
    font-size: 10px;
    width: 12px;
    text-align: center;
    transition: transform 0.15s;
    user-select: none;
    line-height: 1.6;
  }

  .log-entry.expanded .log-chevron { transform: rotate(90deg); color: #888; }

  .log-time {
    color: #555;
    flex-shrink: 0;
    font-size: 11px;
    min-width: 85px;
  }

  .log-level {
    flex-shrink: 0;
    font-weight: 600;
    font-size: 10px;
    padding: 1px 5px;
    border-radius: 2px;
    min-width: 46px;
    text-align: center;
  }

  .log-level.DEBUG { background: #2e1065; color: #a78bfa; }
  .log-level.INFO  { background: #172554; color: #60a5fa; }
  .log-level.WARN  { background: #422006; color: #fbbf24; }
  .log-level.ERROR { background: #450a0a; color: #f87171; }
  .log-level.FATAL { background: #7f1d1d; color: #fca5a5; }

  .log-message {
    flex: 1;
    min-width: 0;
    word-wrap: break-word;
    overflow-wrap: break-word;
    color: #d4d4d4;
  }

  .log-source {
    color: #555;
    flex-shrink: 0;
    font-size: 10px;
    text-align: right;
    max-width: 280px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .log-source .fn { color: #888; }

  .log-copy-btn {
    flex-shrink: 0;
    background: none;
    border: 1px solid transparent;
    color: #444;
    font-family: inherit;
    font-size: 10px;
    cursor: pointer;
    padding: 1px 5px;
    border-radius: 3px;
    transition: all 0.15s;
    opacity: 0;
  }

  .log-entry:hover .log-copy-btn { opacity: 1; }
  .log-copy-btn:hover { color: #aaa; border-color: #444; background: #252525; }
  .log-copy-btn.copied { color: #4ade80; }

  .log-details {
    width: 100%;
    display: none;
    padding: 6px 0 6px 22px;
  }

  .log-entry.expanded .log-details { display: block; }

  .log-detail-row {
    display: flex;
    gap: 8px;
    font-size: 10px;
    padding: 2px 0;
    min-width: 0;
  }

  .log-detail-label {
    color: #555;
    flex-shrink: 0;
    min-width: 60px;
    text-align: right;
  }

  .log-detail-value {
    color: #d4d4d4;
    word-wrap: break-word;
    overflow-wrap: break-word;
    min-width: 0;
    flex: 1;
  }

  .log-detail-value.field-key { color: #a78bfa; }
  .log-detail-value.field-val { color: #86efac; }

  .log-fields-detail {
    font-size: 10px;
    padding: 4px 0;
  }

  .log-field-pair {
    display: flex;
    gap: 4px;
    padding: 1px 0;
    min-width: 0;
    flex-wrap: wrap;
  }

  .stack-trace {
    font-size: 10px;
    color: #f87171;
    margin-top: 4px;
    white-space: pre-wrap;
    word-wrap: break-word;
    opacity: 0.8;
    max-height: 200px;
    overflow-y: auto;
  }

  #footer {
    background: #161616;
    border-top: 1px solid #2a2a2a;
    padding: 6px 16px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-shrink: 0;
    font-size: 11px;
    color: #666;
  }

  #footer .counters {
    display: flex;
    gap: 12px;
    align-items: center;
  }

  .counter {
    display: flex;
    align-items: center;
    gap: 4px;
    font-weight: 600;
  }

  .counter.errors { color: #f87171; }
  .counter.warnings { color: #fbbf24; }
  .counter .count-badge {
    background: #252525;
    padding: 1px 6px;
    border-radius: 3px;
    min-width: 20px;
    text-align: center;
  }

  .counter.errors .count-badge { background: #450a0a; }
  .counter.warnings .count-badge { background: #422006; }

  .new-entry-flash {
    animation: flash 0.3s ease;
  }

  @keyframes flash {
    0% { background: rgba(96, 165, 250, 0.15); }
    100% { background: transparent; }
  }

  .copy-toast {
    position: fixed;
    bottom: 40px;
    left: 50%;
    transform: translateX(-50%);
    background: #252525;
    border: 1px solid #444;
    color: #e0e0e0;
    padding: 6px 16px;
    border-radius: 6px;
    font-size: 11px;
    opacity: 0;
    transition: opacity 0.2s;
    pointer-events: none;
    z-index: 100;
  }

  .copy-toast.visible { opacity: 1; }
</style>
</head>
<body>

<div id="header">
  <h1><span class="dot" id="status-dot"></span> blight debug console</h1>
  <div class="header-actions">
    <button onclick="copyAllLogs()">⎘ Copy All</button>
    <button onclick="scrollToBottom()">↓ Bottom</button>
    <button onclick="clearLogs()">Clear</button>
    <button onclick="toggleAutoScroll()" id="autoscroll-btn">Auto-scroll: ON</button>
  </div>
</div>

<div id="filter-bar">
  <button class="filter-btn active" data-level="all" onclick="toggleFilter('all')">All</button>
  <button class="filter-btn level-debug active" data-level="DEBUG" onclick="toggleFilter('DEBUG')">Debug</button>
  <button class="filter-btn level-info active" data-level="INFO" onclick="toggleFilter('INFO')">Info</button>
  <button class="filter-btn level-warn active" data-level="WARN" onclick="toggleFilter('WARN')">Warn</button>
  <button class="filter-btn level-error active" data-level="ERROR" onclick="toggleFilter('ERROR')">Error</button>
  <input type="text" id="search-input" placeholder="Filter logs..." oninput="applyFilters()">
</div>

<div id="log-container"></div>
<div class="copy-toast" id="copy-toast">Copied!</div>

<div id="footer">
  <span id="log-count">0 entries</span>
  <div class="counters">
    <div class="counter warnings">
      ⚠ <span class="count-badge" id="warn-count">0</span>
    </div>
    <div class="counter errors">
      ✕ <span class="count-badge" id="error-count">0</span>
    </div>
  </div>
</div>

<script>
const container = document.getElementById('log-container');
const logCountEl = document.getElementById('log-count');
const warnCountEl = document.getElementById('warn-count');
const errorCountEl = document.getElementById('error-count');
const statusDot = document.getElementById('status-dot');
const searchInput = document.getElementById('search-input');

let entries = [];
let autoScroll = true;
let activeFilters = new Set(['DEBUG', 'INFO', 'WARN', 'ERROR', 'FATAL']);
let errorCount = 0;
let warnCount = 0;

function addEntry(entry) {
  entries.push(entry);
  if (entry.level === 'ERROR' || entry.level === 'FATAL') errorCount++;
  if (entry.level === 'WARN') warnCount++;
  updateCounters();

  if (isVisible(entry)) {
    const el = renderEntry(entry);
    container.appendChild(el);
    if (autoScroll) el.scrollIntoView({ behavior: 'smooth' });
  }
}

function renderEntry(entry) {
  const div = document.createElement('div');
  div.className = 'log-entry level-' + entry.level.toLowerCase() + ' new-entry-flash';
  div.dataset.entryIndex = entries.indexOf(entry);
  setTimeout(() => div.classList.remove('new-entry-flash'), 300);

  const hasDetails = (entry.fields && Object.keys(entry.fields).length > 0) || entry.function || entry.file;

  // Build detail panel (collapsed by default)
  let detailsHTML = '<div class="log-details">';
  // Source info
  if (entry.function || entry.file) {
    detailsHTML += '<div class="log-detail-row"><span class="log-detail-label">source</span><span class="log-detail-value">' + escapeHTML(entry.function) + '</span></div>';
    detailsHTML += '<div class="log-detail-row"><span class="log-detail-label">file</span><span class="log-detail-value">' + escapeHTML(entry.file + ':' + entry.line) + '</span></div>';
  }
  detailsHTML += '<div class="log-detail-row"><span class="log-detail-label">time</span><span class="log-detail-value">' + escapeHTML(entry.time) + '</span></div>';
  // Fields
  if (entry.fields && Object.keys(entry.fields).length > 0) {
    const stack = entry.fields.stack;
    const otherFields = Object.entries(entry.fields).filter(([k]) => k !== 'stack');
    if (otherFields.length > 0) {
      detailsHTML += '<div class="log-fields-detail">';
      otherFields.forEach(([k, v]) => {
        const val = typeof v === 'string' ? v : JSON.stringify(v);
        detailsHTML += '<div class="log-field-pair"><span class="log-detail-value field-key">' + escapeHTML(k) + '</span><span class="log-detail-value field-val">' + escapeHTML(val) + '</span></div>';
      });
      detailsHTML += '</div>';
    }
    if (stack) {
      detailsHTML += '<div class="stack-trace">' + escapeHTML(stack) + '</div>';
    }
  }
  detailsHTML += '</div>';

  // Inline field preview (shown in collapsed state)
  let inlineFields = '';
  if (entry.fields) {
    const otherFields = Object.entries(entry.fields).filter(([k]) => k !== 'stack');
    if (otherFields.length > 0) {
      const preview = otherFields.map(([k, v]) => k + '=' + (typeof v === 'string' ? v : JSON.stringify(v))).join('  ');
      const truncated = preview.length > 80 ? preview.substring(0, 80) + '…' : preview;
      inlineFields = ' <span style="color:#555;font-size:10px">' + escapeHTML(truncated) + '</span>';
    }
  }

  div.innerHTML =
    '<div class="log-summary">' +
      (hasDetails ? '<span class="log-chevron">▶</span>' : '<span class="log-chevron" style="visibility:hidden">▶</span>') +
      '<span class="log-time">' + entry.time + '</span>' +
      '<span class="log-level ' + entry.level + '">' + entry.level + '</span>' +
      '<span class="log-message">' + escapeHTML(entry.message) + inlineFields + '</span>' +
      '<button class="log-copy-btn" onclick="copyEntry(event, ' + entries.indexOf(entry) + ')" title="Copy log">⎘</button>' +
    '</div>' +
    detailsHTML;

  // Click to expand/collapse
  div.querySelector('.log-summary').addEventListener('click', (e) => {
    if (e.target.classList.contains('log-copy-btn')) return;
    div.classList.toggle('expanded');
  });

  return div;
}

function copyEntry(event, index) {
  event.stopPropagation();
  const entry = entries[index];
  if (!entry) return;

  let text = '[' + entry.time + '] [' + entry.level + '] ' + entry.message;
  if (entry.function) text += '\n  source: ' + entry.function;
  if (entry.file) text += '\n  file: ' + entry.file + ':' + entry.line;
  if (entry.fields) {
    Object.entries(entry.fields).forEach(([k, v]) => {
      text += '\n  ' + k + ': ' + (typeof v === 'string' ? v : JSON.stringify(v));
    });
  }

  navigator.clipboard.writeText(text);
  const btn = event.target;
  btn.classList.add('copied');
  btn.textContent = '✓';
  setTimeout(() => { btn.classList.remove('copied'); btn.textContent = '⎘'; }, 1000);
  showCopyToast('Log entry copied');
}

function copyAllLogs() {
  const visible = entries.filter(e => isVisible(e));
  const text = visible.map(entry => {
    let line = '[' + entry.time + '] [' + entry.level + '] ' + entry.message;
    if (entry.function) line += '  (' + entry.function + ' ' + entry.file + ':' + entry.line + ')';
    if (entry.fields) {
      const fieldPairs = Object.entries(entry.fields)
        .filter(([k]) => k !== 'stack')
        .map(([k, v]) => k + '=' + (typeof v === 'string' ? v : JSON.stringify(v)));
      if (fieldPairs.length > 0) line += '\n  ' + fieldPairs.join('  ');
      if (entry.fields.stack) line += '\n  ' + entry.fields.stack;
    }
    return line;
  }).join('\n');

  navigator.clipboard.writeText(text);
  showCopyToast(visible.length + ' log entries copied');
}

function showCopyToast(message) {
  const toast = document.getElementById('copy-toast');
  toast.textContent = message;
  toast.classList.add('visible');
  setTimeout(() => toast.classList.remove('visible'), 1500);
}

function isVisible(entry) {
  if (!activeFilters.has(entry.level)) return false;
  const query = searchInput.value.toLowerCase();
  if (query && !entry.message.toLowerCase().includes(query) &&
      !entry.function.toLowerCase().includes(query) &&
      !entry.file.toLowerCase().includes(query)) return false;
  return true;
}

function updateCounters() {
  logCountEl.textContent = entries.length + ' entries';
  warnCountEl.textContent = warnCount;
  errorCountEl.textContent = errorCount;
}

function rerenderAll() {
  container.innerHTML = '';
  entries.forEach(entry => {
    if (isVisible(entry)) {
      container.appendChild(renderEntry(entry));
    }
  });
  if (autoScroll) scrollToBottom();
}

function toggleFilter(level) {
  if (level === 'all') {
    const allActive = activeFilters.size >= 5;
    if (allActive) {
      activeFilters.clear();
    } else {
      activeFilters = new Set(['DEBUG', 'INFO', 'WARN', 'ERROR', 'FATAL']);
    }
  } else {
    if (activeFilters.has(level)) activeFilters.delete(level);
    else activeFilters.add(level);
  }

  document.querySelectorAll('.filter-btn').forEach(btn => {
    const lvl = btn.dataset.level;
    if (lvl === 'all') {
      btn.classList.toggle('active', activeFilters.size >= 5);
    } else {
      btn.classList.toggle('active', activeFilters.has(lvl));
    }
  });

  rerenderAll();
}

function applyFilters() {
  rerenderAll();
}

function clearLogs() {
  entries = [];
  errorCount = 0;
  warnCount = 0;
  container.innerHTML = '';
  updateCounters();
}

function scrollToBottom() {
  container.scrollTop = container.scrollHeight;
}

function toggleAutoScroll() {
  autoScroll = !autoScroll;
  document.getElementById('autoscroll-btn').textContent = 'Auto-scroll: ' + (autoScroll ? 'ON' : 'OFF');
}

function escapeHTML(str) {
  const div = document.createElement('div');
  div.textContent = str;
  return div.innerHTML;
}

// Load history then connect SSE
fetch('/api/history')
  .then(r => r.json())
  .then(data => {
    data.forEach(e => addEntry(e));
    scrollToBottom();
    connectSSE();
  })
  .catch(() => connectSSE());

function connectSSE() {
  const es = new EventSource('/api/stream');

  es.onopen = () => {
    statusDot.classList.remove('disconnected');
  };

  es.onmessage = (e) => {
    try {
      const entry = JSON.parse(e.data);
      addEntry(entry);
    } catch {}
  };

  es.onerror = () => {
    statusDot.classList.add('disconnected');
    es.close();
    setTimeout(connectSSE, 2000);
  };
}
</script>
</body>
</html>`

func StartConsole(logger *Logger) (int, error) {
	if !logger.Enabled() {
		return 0, nil
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(consoleHTML))
	})

	mux.HandleFunc("/api/history", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		history := logger.History()
		if history == nil {
			history = []LogEntry{}
		}
		json.NewEncoder(w).Encode(history)
	})

	mux.HandleFunc("/api/stream", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		ch := logger.Subscribe()
		defer logger.Unsubscribe(ch)

		// Keep alive
		ctx := r.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case entry, ok := <-ch:
				if !ok {
					return
				}
				data, _ := json.Marshal(entry)
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			}
		}
	})

	// Find a free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}

	port := listener.Addr().(*net.TCPAddr).Port

	go http.Serve(listener, mux)

	return port, nil
}

func OpenInBrowser(port int) {
	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	exec.Command("cmd", "/c", "start", "", strings.ReplaceAll(url, "&", "^&")).Start()
}
