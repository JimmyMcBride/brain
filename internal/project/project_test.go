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

func TestLookupParadigm(t *testing.T) {
	p, err := LookupParadigm("epics")
	if err != nil {
		t.Fatal(err)
	}
	if p.ContainerType != "epic" || p.ItemType != "story" {
		t.Fatalf("unexpected paradigm: %+v", p)
	}
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
	if info.Paradigm != nil {
		t.Fatal("expected nil paradigm before plan init")
	}
}

func TestInitWritesProjectConfig(t *testing.T) {
	mgr, root := setupProjectManager(t)
	info, err := mgr.Init("epics")
	if err != nil {
		t.Fatal(err)
	}
	if info.Paradigm == nil || info.Paradigm.Name != "epics" {
		t.Fatalf("unexpected paradigm: %+v", info.Paradigm)
	}
	for _, rel := range []string{
		".brain/project.yaml",
		".brain/brainstorms",
		".brain/planning/epics",
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
	if _, err := mgr.Init("epics"); err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Init("cycles"); err == nil {
		t.Fatal("expected second init to fail")
	}
}
