package journal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Entry represents a single journal entry
type Entry struct {
	Timestamp time.Time              `json:"timestamp"`
	Action    string                 `json:"action"`
	Status    string                 `json:"status"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Duration  int64                  `json:"duration_ms,omitempty"`
}

// Logger is an append-only journal logger
type Logger struct {
	journalDir string
	mu         sync.Mutex
	indexMu    sync.Mutex
}

// NewLogger creates a new journal logger
func NewLogger(journalDir string) (*Logger, error) {
	// Create directory if not exists
	if err := os.MkdirAll(journalDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create journal directory: %w", err)
	}

	return &Logger{
		journalDir: journalDir,
	}, nil
}

// Log appends an entry to the journal
func (l *Logger) Log(action, status string, details map[string]interface{}, err error) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := Entry{
		Timestamp: time.Now(),
		Action:    action,
		Status:    status,
		Details:   details,
	}

	if err != nil {
		entry.Error = err.Error()
	}

	// Get today's log file
	logFile := l.getLogFile()

	// Append to file (raw text format)
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	// Format: timestamp [ACTION] key1=val1 key2=val2
	line := l.formatEntry(&entry)
	if _, err := f.WriteString(line + "\n"); err != nil {
		return fmt.Errorf("failed to write to log: %w", err)
	}

	// Update index
	if err := l.updateIndex(&entry); err != nil {
		// Don't fail on index error, just log it
		fmt.Fprintf(os.Stderr, "warning: failed to update index: %v\n", err)
	}

	return nil
}

// LogMigration logs a migration event
func (l *Logger) LogMigration(version string, status string, duration int64, backupPath string, details map[string]interface{}) error {
	if details == nil {
		details = make(map[string]interface{})
	}
	details["version"] = version
	details["backup_path"] = backupPath

	entry := Entry{
		Timestamp: time.Now(),
		Action:    "migrate",
		Status:    status,
		Details:   details,
		Duration:  duration,
	}

	return l.logEntry(&entry)
}

// LogSchema logs a schema event
func (l *Logger) LogSchema(action string, status string, details map[string]interface{}) error {
	return l.Log(action, status, details, nil)
}

// LogError logs an error event
func (l *Logger) LogError(action string, err error, details map[string]interface{}) error {
	return l.Log(action, "error", details, err)
}

// logEntry writes entry to file and updates index
func (l *Logger) logEntry(entry *Entry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	logFile := l.getLogFile()

	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	line := l.formatEntry(entry)
	if _, err := f.WriteString(line + "\n"); err != nil {
		return fmt.Errorf("failed to write to log: %w", err)
	}

	// Update index
	if err := l.updateIndex(entry); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to update index: %v\n", err)
	}

	return nil
}

// formatEntry formats an entry as Unix-style log line
// Format: 2026-02-12T10:15:00Z [ACTION] key1=val1 key2=val2 error="msg"
func (l *Logger) formatEntry(e *Entry) string {
	line := fmt.Sprintf("%s [%s] status=%s", e.Timestamp.Format(time.RFC3339), e.Action, e.Status)

	// Add details
	for k, v := range e.Details {
		line += fmt.Sprintf(" %s=%v", k, v)
	}

	// Add duration if present
	if e.Duration > 0 {
		line += fmt.Sprintf(" duration_ms=%d", e.Duration)
	}

	// Add error if present
	if e.Error != "" {
		line += fmt.Sprintf(" error=%q", e.Error)
	}

	return line
}

// getLogFile returns the path to today's log file
func (l *Logger) getLogFile() string {
	today := time.Now().Format("2006-01-02")
	return filepath.Join(l.journalDir, today+".log")
}

// updateIndex updates the daily index
func (l *Logger) updateIndex(e *Entry) error {
	l.indexMu.Lock()
	defer l.indexMu.Unlock()

	indexFile := filepath.Join(l.journalDir, "index.json")

	// Load existing index
	var index map[string]interface{}
	if data, err := os.ReadFile(indexFile); err == nil {
		if err := json.Unmarshal(data, &index); err != nil {
			return err
		}
	} else {
		index = make(map[string]interface{})
	}

	// Update metadata
	today := time.Now().Format("2006-01-02")
	if index["date"] != today {
		index["date"] = today
		index["entries"] = 0
		index["by_action"] = make(map[string]interface{})
	}

	// Increment counters - handle float64 from JSON
	var entries int
	if val, exists := index["entries"]; exists {
		if floatVal, ok := val.(float64); ok {
			entries = int(floatVal)
		} else if intVal, ok := val.(int); ok {
			entries = intVal
		}
	}
	index["entries"] = entries + 1

	// Update by_action map
	byAction, ok := index["by_action"].(map[string]interface{})
	if !ok {
		byAction = make(map[string]interface{})
		index["by_action"] = byAction
	}

	count := byAction[e.Action]
	if count == nil {
		byAction[e.Action] = 1
	} else {
		// Handle both int and float64 from JSON
		var countVal int
		if floatVal, ok := count.(float64); ok {
			countVal = int(floatVal)
		} else if intVal, ok := count.(int); ok {
			countVal = intVal
		}
		byAction[e.Action] = countVal + 1
	}

	// Save index
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(indexFile, data, 0644)
}

// Last returns the last N entries from today's log
func (l *Logger) Last(n int) ([]*Entry, error) {
	logFile := l.getLogFile()

	data, err := os.ReadFile(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Entry{}, nil
		}
		return nil, err
	}

	// Parse lines
	lines := strings.Split(string(data), "\n")

	var entries []*Entry
	for _, line := range lines {
		if line == "" {
			continue
		}
		entry, err := l.parseEntry(line)
		if err != nil {
			// Skip unparseable lines
			continue
		}
		entries = append(entries, entry)
	}

	// Return last N
	if len(entries) > n {
		entries = entries[len(entries)-n:]
	}

	return entries, nil
}

// Errors returns all error entries from today
func (l *Logger) Errors() ([]*Entry, error) {
	logFile := l.getLogFile()

	data, err := os.ReadFile(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Entry{}, nil
		}
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	var entries []*Entry

	for _, line := range lines {
		if line == "" || !strings.Contains(line, "status=error") {
			continue
		}
		entry, err := l.parseEntry(line)
		if err != nil {
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// Migrations returns all migration entries
func (l *Logger) Migrations() ([]*Entry, error) {
	logFile := l.getLogFile()

	data, err := os.ReadFile(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Entry{}, nil
		}
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	var entries []*Entry

	for _, line := range lines {
		if line == "" || !strings.Contains(line, "[migrate]") {
			continue
		}
		entry, err := l.parseEntry(line)
		if err != nil {
			continue
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// parseEntry parses a log line into an Entry
// Format: 2026-02-12T10:15:00Z [ACTION] status=ok key1=val1 key2=val2
func (l *Logger) parseEntry(line string) (*Entry, error) {
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid log line format")
	}

	timestamp, err := time.Parse(time.RFC3339, parts[0])
	if err != nil {
		return nil, err
	}

	// Extract action from [ACTION]
	actionStr := parts[1]
	action := strings.Trim(actionStr, "[]")

	entry := &Entry{
		Timestamp: timestamp,
		Action:    action,
	}

	// Parse remaining fields
	for _, part := range parts[2:] {
		if strings.HasPrefix(part, "status=") {
			entry.Status = strings.TrimPrefix(part, "status=")
		} else if strings.HasPrefix(part, "error=") {
			errVal := strings.TrimPrefix(part, "error=")
			entry.Error = strings.Trim(errVal, "\"")
		}
	}

	return entry, nil
}
