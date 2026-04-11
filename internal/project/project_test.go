package project

import (
	"os"
	"path/filepath"
	"testing"

	"brain/internal/backup"
	"brain/internal/history"
	"brain/internal/notes"
	"brain/internal/templates"
	"brain/internal/workspace"
)

func setupProjectManager(t *testing.T) (*Manager, string) {
	t.Helper()
	root := t.TempDir()
	stateDir := filepath.Join(root, ".brain", "state")
	workspaceSvc := workspace.New(root)
	if err := workspaceSvc.Initialize(); err != nil {
		t.Fatal(err)
	}
	nm := notes.New(
		workspaceSvc,
		templates.New(),
		backup.New(filepath.Join(stateDir, "backups")),
		history.New(filepath.Join(stateDir, "history.jsonl")),
	)
	return New(nm, workspaceSvc), root
}

func TestResolveWithoutPlanningInit(t *testing.T) {
	mgr, root := setupProjectManager(t)
	info, err := mgr.Resolve()
	if err != nil {
		t.Fatal(err)
	}
	if info.Name != filepath.Base(root) {
		t.Fatalf("unexpected project name: %s", info.Name)
	}
	if info.PlanningInitialized {
		t.Fatal("expected planning to be uninitialized before plan init")
	}
}

func TestInitWritesProjectConfig(t *testing.T) {
	mgr, root := setupProjectManager(t)
	info, err := mgr.Init()
	if err != nil {
		t.Fatal(err)
	}
	if !info.PlanningInitialized || info.PlanningModel != "epic_spec_v1" {
		t.Fatalf("unexpected planning state: %+v", info)
	}
	for _, rel := range []string{
		".brain/project.yaml",
		".brain/brainstorms",
		".brain/planning/epics",
		".brain/planning/specs",
		".brain/planning/stories",
		".brain/resources/captures",
	} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("missing expected path %s: %v", rel, err)
		}
	}
}

func TestInitTwiceFails(t *testing.T) {
	mgr, _ := setupProjectManager(t)
	if _, err := mgr.Init(); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Init(); err == nil {
		t.Fatal("expected second init to fail")
	}
}

func TestResolveRejectsUnknownPlanningModel(t *testing.T) {
	mgr, root := setupProjectManager(t)
	if err := os.WriteFile(filepath.Join(root, ".brain", "project.yaml"), []byte("name: test\nplanning_model: unknown\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Resolve(); err == nil {
		t.Fatal("expected unknown planning model to fail")
	}
}
