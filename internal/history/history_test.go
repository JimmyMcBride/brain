package history

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"brain/internal/backup"
	"brain/internal/config"
	"brain/internal/vault"
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
	cfg := &config.Config{VaultPath: filepath.Join(root, "vault"), DataPath: filepath.Join(root, "data")}
	vaultSvc := vault.New(cfg)
	if err := vaultSvc.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cfg.DataPath, 0o755); err != nil {
		t.Fatal(err)
	}
	backups := backup.New(filepath.Join(cfg.DataPath, "backups"))
	logger := New(filepath.Join(cfg.DataPath, "history.jsonl"))
	undoer := NewUndoer(logger, backups, vaultSvc)

	path := filepath.Join(cfg.VaultPath, "Resources", "note.md")
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
	if err := logger.Append(Entry{ID: "update1", Operation: "update", File: "Resources/note.md", BackupPath: backupPath, Summary: "updated"}); err != nil {
		t.Fatal(err)
	}

	if _, err := undoer.Undo(); err != nil {
		t.Fatal(err)
	}
	restored, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(restored) != "old" {
		t.Fatalf("expected restored content, got %q", restored)
	}

	moveBackup, err := backups.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(cfg.VaultPath, "Areas", "note.md")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(path, dest); err != nil {
		t.Fatal(err)
	}
	if err := logger.Append(Entry{ID: "move1", Operation: "move", File: "Resources/note.md", Target: "Areas/note.md", BackupPath: moveBackup, Summary: "moved"}); err != nil {
		t.Fatal(err)
	}
	if _, err := undoer.Undo(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected original file restored: %v", err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("expected destination removed, got %v", err)
	}
	all, err := logger.All()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) == 0 || all[len(all)-1].Operation != "undo" {
		t.Fatalf("expected undo entry appended: %+v", all)
	}
	if !strings.Contains(all[len(all)-1].Summary, "reverted move") {
		t.Fatalf("unexpected undo summary: %+v", all[len(all)-1])
	}
}

func TestUndoCreateRemovesFile(t *testing.T) {
	root := t.TempDir()
	cfg := &config.Config{VaultPath: filepath.Join(root, "vault"), DataPath: filepath.Join(root, "data")}
	vaultSvc := vault.New(cfg)
	if err := vaultSvc.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cfg.DataPath, 0o755); err != nil {
		t.Fatal(err)
	}
	logger := New(filepath.Join(cfg.DataPath, "history.jsonl"))
	undoer := NewUndoer(logger, backup.New(filepath.Join(cfg.DataPath, "backups")), vaultSvc)
	path := filepath.Join(cfg.VaultPath, "Resources", "created.md")
	if err := os.WriteFile(path, []byte("created"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := logger.Append(Entry{ID: "create1", Operation: "create", File: "Resources/created.md", Summary: "created"}); err != nil {
		t.Fatal(err)
	}
	if _, err := undoer.Undo(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected file removed, got %v", err)
	}
}
