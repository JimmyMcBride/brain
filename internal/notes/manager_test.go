package notes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"brain/internal/backup"
	"brain/internal/history"
	"brain/internal/templates"
	"brain/internal/workspace"
)

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	root := t.TempDir()
	stateDir := filepath.Join(root, ".brain", "state")
	workspaceSvc := workspace.New(root)
	if err := workspaceSvc.Initialize(); err != nil {
		t.Fatal(err)
	}
	return New(
		workspaceSvc,
		templates.New(),
		backup.New(filepath.Join(stateDir, "backups")),
		history.New(filepath.Join(stateDir, "history.jsonl")),
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
	if meta["title"] != "Alpha" || strings.TrimSpace(body) != "# Body" {
		t.Fatalf("unexpected parse result: %+v %q", meta, body)
	}
}

func TestManagerLifecycle(t *testing.T) {
	manager := newTestManager(t)
	note, err := manager.Create(CreateInput{
		Title:    "Alpha Note",
		NoteType: "resource",
		Section:  ".brain",
		Subdir:   "resources/references",
		Metadata: map[string]any{"topic": "alpha"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if note.Path != ".brain/resources/references/alpha-note.md" {
		t.Fatalf("unexpected path: %s", note.Path)
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
	if oldPath != ".brain/resources/references/alpha-note.md" || newPath != ".brain/resources/references/beta-note.md" {
		t.Fatalf("unexpected rename: %s -> %s", oldPath, newPath)
	}

	_, movedPath, err := manager.Move(newPath, "docs/")
	if err != nil {
		t.Fatal(err)
	}
	if movedPath != "docs/beta-note.md" {
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
