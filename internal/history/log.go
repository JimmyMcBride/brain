package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type Entry struct {
	ID         string         `json:"id"`
	Timestamp  time.Time      `json:"timestamp"`
	Operation  string         `json:"operation"`
	File       string         `json:"file"`
	Target     string         `json:"target,omitempty"`
	BackupPath string         `json:"backup_path,omitempty"`
	Summary    string         `json:"summary"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	UndoOf     string         `json:"undo_of,omitempty"`
}

type Logger struct {
	Path string
}

func New(path string) *Logger {
	return &Logger{Path: path}
}

func (l *Logger) Append(entry Entry) error {
	if entry.ID == "" {
		entry.ID = fmt.Sprintf("%d", time.Now().UTC().UnixNano())
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
	f, err := os.OpenFile(l.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open history log: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	if err := enc.Encode(entry); err != nil {
		return fmt.Errorf("append history entry: %w", err)
	}
	return nil
}

func (l *Logger) List(limit int) ([]Entry, error) {
	all, err := l.All()
	if err != nil {
		return nil, err
	}
	if limit > 0 && limit < len(all) {
		all = all[len(all)-limit:]
	}
	for i, j := 0, len(all)-1; i < j; i, j = i+1, j-1 {
		all[i], all[j] = all[j], all[i]
	}
	return all, nil
}

func (l *Logger) All() ([]Entry, error) {
	f, err := os.OpenFile(l.Path, os.O_CREATE|os.O_RDONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open history log: %w", err)
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry Entry
		if err := json.Unmarshal(line, &entry); err != nil {
			return nil, fmt.Errorf("parse history entry: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan history log: %w", err)
	}
	return entries, nil
}
