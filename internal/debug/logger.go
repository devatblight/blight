package debug

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Level string

const (
	LevelDebug Level = "DEBUG"
	LevelInfo  Level = "INFO"
	LevelWarn  Level = "WARN"
	LevelError Level = "ERROR"
	LevelFatal Level = "FATAL"
)

type LogEntry struct {
	Time     string                 `json:"time"`
	Level    Level                  `json:"level"`
	Message  string                 `json:"message"`
	File     string                 `json:"file"`
	Line     int                    `json:"line"`
	Function string                 `json:"function"`
	Fields   map[string]interface{} `json:"fields,omitempty"`
}

type Logger struct {
	mu         sync.Mutex
	file       *os.File
	entries    []LogEntry
	sseClients []chan LogEntry
	enabled    bool
	logPath    string
}

var instance *Logger
var once sync.Once

func Init() *Logger {
	once.Do(func() {
		env := os.Getenv("BLIGHT_ENV")
		enabled := env != "production"

		home, _ := os.UserHomeDir()
		logDir := filepath.Join(home, ".blight")
		os.MkdirAll(logDir, 0755)
		logPath := filepath.Join(logDir, "debug.log")

		var logFile *os.File
		if enabled {
			logFile, _ = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)

			// Write session header
			if logFile != nil {
				header := fmt.Sprintf("\n═══════════════════════════════════════════════════════\n"+
					"  BLIGHT DEBUG SESSION — %s\n"+
					"  PID: %d\n"+
					"═══════════════════════════════════════════════════════\n\n",
					time.Now().Format("2006-01-02 15:04:05"), os.Getpid())
				logFile.WriteString(header)
			}
		}

		instance = &Logger{
			enabled: enabled,
			file:    logFile,
			logPath: logPath,
		}
	})
	return instance
}

func Get() *Logger {
	if instance == nil {
		return Init()
	}
	return instance
}

func (l *Logger) Enabled() bool {
	return l.enabled
}

func (l *Logger) LogPath() string {
	return l.logPath
}

func (l *Logger) Close() {
	if l.file != nil {
		l.file.Close()
	}
}

func (l *Logger) Subscribe() chan LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	ch := make(chan LogEntry, 100)
	l.sseClients = append(l.sseClients, ch)
	return ch
}

func (l *Logger) Unsubscribe(ch chan LogEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i, c := range l.sseClients {
		if c == ch {
			l.sseClients = append(l.sseClients[:i], l.sseClients[i+1:]...)
			close(ch)
			return
		}
	}
}

func (l *Logger) History() []LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	result := make([]LogEntry, len(l.entries))
	copy(result, l.entries)
	return result
}

func (l *Logger) log(level Level, msg string, fields map[string]interface{}, skip int) {
	if !l.enabled {
		return
	}

	pc, file, line, ok := runtime.Caller(skip)
	funcName := "unknown"
	if ok {
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			funcName = fn.Name()
			// Shorten to just package.Function
			parts := strings.Split(funcName, "/")
			funcName = parts[len(parts)-1]
		}
		// Shorten file path
		parts := strings.Split(filepath.ToSlash(file), "/")
		if len(parts) > 3 {
			file = strings.Join(parts[len(parts)-3:], "/")
		}
	}

	entry := LogEntry{
		Time:     time.Now().Format("15:04:05.000"),
		Level:    level,
		Message:  msg,
		File:     file,
		Line:     line,
		Function: funcName,
		Fields:   fields,
	}

	l.mu.Lock()
	l.entries = append(l.entries, entry)
	// Keep max 2000 entries in memory
	if len(l.entries) > 2000 {
		l.entries = l.entries[len(l.entries)-2000:]
	}

	// Write to file (survives crashes since we flush immediately)
	if l.file != nil {
		levelPad := string(level)
		for len(levelPad) < 5 {
			levelPad += " "
		}
		line := fmt.Sprintf("[%s] %s  %s  (%s:%d %s)",
			entry.Time, levelPad, entry.Message, entry.File, entry.Line, entry.Function)
		if len(fields) > 0 {
			fieldsJSON, _ := json.Marshal(fields)
			line += "  " + string(fieldsJSON)
		}
		l.file.WriteString(line + "\n")
		l.file.Sync() // Flush immediately — critical for crash resilience
	}

	// Broadcast to SSE clients
	for _, ch := range l.sseClients {
		select {
		case ch <- entry:
		default:
			// Drop if client is full
		}
	}
	l.mu.Unlock()
}

func (l *Logger) Debug(msg string, fields ...map[string]interface{}) {
	f := mergeFields(fields)
	l.log(LevelDebug, msg, f, 2)
}

func (l *Logger) Info(msg string, fields ...map[string]interface{}) {
	f := mergeFields(fields)
	l.log(LevelInfo, msg, f, 2)
}

func (l *Logger) Warn(msg string, fields ...map[string]interface{}) {
	f := mergeFields(fields)
	l.log(LevelWarn, msg, f, 2)
}

func (l *Logger) Error(msg string, fields ...map[string]interface{}) {
	f := mergeFields(fields)
	l.log(LevelError, msg, f, 2)
}

func (l *Logger) Fatal(msg string, fields ...map[string]interface{}) {
	f := mergeFields(fields)
	l.log(LevelFatal, msg, f, 2)
}

// RecoverPanic should be deferred at the top of main() and goroutines
func (l *Logger) RecoverPanic(context string) {
	if r := recover(); r != nil {
		buf := make([]byte, 4096)
		n := runtime.Stack(buf, false)
		stackTrace := string(buf[:n])

		l.log(LevelFatal, fmt.Sprintf("PANIC in %s: %v", context, r), map[string]interface{}{
			"panic": fmt.Sprintf("%v", r),
			"stack": stackTrace,
		}, 3)

		// Also write directly to file in case the logger is broken
		if l.file != nil {
			l.file.WriteString(fmt.Sprintf("\n!!! PANIC in %s: %v\n%s\n", context, r, stackTrace))
			l.file.Sync()
		}
	}
}

func mergeFields(fields []map[string]interface{}) map[string]interface{} {
	if len(fields) == 0 {
		return nil
	}
	return fields[0]
}
