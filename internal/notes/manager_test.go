package notes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"brain/internal/backup"
	"brain/internal/config"
	"brain/internal/history"
	"brain/internal/templates"
	"brain/internal/vault"
)

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	root := t.TempDir()
	cfg := &config.Config{
		VaultPath: filepath.Join(root, "vault"),
		DataPath:  filepath.Join(root, "data"),
	}
	vaultSvc := vault.New(cfg)
	if err := vaultSvc.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cfg.DataPath, 0o755); err != nil {
		t.Fatal(err)
	}
	return New(
		vaultSvc,
		templates.New(),
		backup.New(filepath.Join(cfg.DataPath, "backups")),
		history.New(filepath.Join(cfg.DataPath, "history.jsonl")),
	)
}

func TestFrontmatterRoundTrip(t *testing.T) {
	raw, err := ComposeFrontmatter(map[string]any{"title": "Alpha", "type": "resource"}, "# Body")
	if err != nil {
		t.Fatal(err)
	}
	meta, body, err := ParseFrontmatter(raw)
	if err != nil {
		t.Fatal(err)
	}
	if meta["title"] != "Alpha" {
		t.Fatalf("unexpected metadata: %+v", meta)
	}
	if strings.TrimSpace(body) != "# Body" {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestManagerLifecycle(t *testing.T) {
	manager := newTestManager(t)
	note, err := manager.Create(CreateInput{
		Title:    "Alpha Note",
		NoteType: "resource",
		Section:  "Resources",
		Metadata: map[string]any{"topic": "alpha"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if note.Path != "Resources/alpha-note.md" {
		t.Fatalf("unexpected path: %s", note.Path)
	}
	if _, err := manager.Create(CreateInput{Title: "Alpha Note", Section: "Resources"}); err == nil {
		t.Fatal("expected duplicate create error")
	}

	body := "# Alpha Note\n\nUpdated body."
	updated, err := manager.Update(note.Path, UpdateInput{
		Body:     &body,
		Metadata: map[string]any{"status": "active"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(updated.Content, "Updated body.") {
		t.Fatalf("unexpected content: %s", updated.Content)
	}

	results, err := manager.Find("active", "resource", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 find result, got %d", len(results))
	}

	oldPath, newPath, err := manager.Rename(note.Path, "Beta Note")
	if err != nil {
		t.Fatal(err)
	}
	if oldPath != "Resources/alpha-note.md" || newPath != "Resources/beta-note.md" {
		t.Fatalf("unexpected rename: %s -> %s", oldPath, newPath)
	}

	_, movedPath, err := manager.Move(newPath, "Areas/Reference/")
	if err != nil {
		t.Fatal(err)
	}
	if movedPath != "Areas/Reference/beta-note.md" {
		t.Fatalf("unexpected moved path: %s", movedPath)
	}

	manager.editorRun = func(editor, path string) error {
		return os.WriteFile(path, []byte("---\ntitle: Beta Note\ntype: resource\n---\nEdited in place.\n"), 0o644)
	}
	edited, err := manager.EditInEditor(movedPath, "fake-editor")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(edited.Content) != "Edited in place." {
		t.Fatalf("unexpected edited content: %q", edited.Content)
	}
}
