package history

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"brain/internal/backup"
	"brain/internal/workspace"
)

func TestLoggerListNewestFirst(t *testing.T) {
	log := New(filepath.Join(t.TempDir(), "history.jsonl"))
	entries := []Entry{
		{ID: "1", Timestamp: time.Unix(1, 0).UTC(), Operation: "create", File: "a.md", Summary: "a"},
		{ID: "2", Timestamp: time.Unix(2, 0).UTC(), Operation: "update", File: "b.md", Summary: "b"},
		{ID: "3", Timestamp: time.Unix(3, 0).UTC(), Operation: "move", File: "c.md", Summary: "c"},
	}
	for _, entry := range entries {
		if err := log.Append(entry); err != nil {
			t.Fatal(err)
		}
	}
	got, err := log.List(2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].ID != "3" || got[1].ID != "2" {
		t.Fatalf("unexpected order: %+v", got)
	}
}

func TestUndoUpdateAndMove(t *testing.T) {
	root := t.TempDir()
	stateDir := filepath.Join(root, ".brain", "state")
	workspaceSvc := workspace.New(root)
	if err := workspaceSvc.Initialize(); err != nil {
		t.Fatal(err)
	}
	backups := backup.New(filepath.Join(stateDir, "backups"))
	logger := New(filepath.Join(stateDir, "history.jsonl"))
	undoer := NewUndoer(logger, backups, workspaceSvc)

	path := filepath.Join(root, ".brain", "resources", "references", "note.md")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	backupPath, err := backups.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := logger.Append(Entry{ID: "update1", Operation: "update", File: ".brain/resources/references/note.md", BackupPath: backupPath, Summary: "updated"}); err != nil {
		t.Fatal(err)
	}
	if _, err := undoer.Undo(); err != nil {
		t.Fatal(err)
	}

	moveBackup, err := backups.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(root, "docs", "note.md")
	if err := os.Rename(path, dest); err != nil {
		t.Fatal(err)
	}
	if err := logger.Append(Entry{ID: "move1", Operation: "move", File: ".brain/resources/references/note.md", Target: "docs/note.md", BackupPath: moveBackup, Summary: "moved"}); err != nil {
		t.Fatal(err)
	}
	if _, err := undoer.Undo(); err != nil {
		t.Fatal(err)
	}
	all, err := logger.All()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) == 0 || all[len(all)-1].Operation != "undo" || !strings.Contains(all[len(all)-1].Summary, "reverted move") {
		t.Fatalf("unexpected undo log: %+v", all)
	}
}

func TestUndoCreateRemovesFile(t *testing.T) {
	root := t.TempDir()
	stateDir := filepath.Join(root, ".brain", "state")
	workspaceSvc := workspace.New(root)
	if err := workspaceSvc.Initialize(); err != nil {
		t.Fatal(err)
	}
	logger := New(filepath.Join(stateDir, "history.jsonl"))
	undoer := NewUndoer(logger, backup.New(filepath.Join(stateDir, "backups")), workspaceSvc)
	path := filepath.Join(root, ".brain", "resources", "references", "created.md")
	if err := os.WriteFile(path, []byte("created"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := logger.Append(Entry{ID: "create1", Operation: "create", File: ".brain/resources/references/created.md", Summary: "created"}); err != nil {
		t.Fatal(err)
	}
	if _, err := undoer.Undo(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file removed, got %v", err)
	}
}
