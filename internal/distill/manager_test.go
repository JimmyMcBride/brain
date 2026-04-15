package distill

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"brain/internal/backup"
	"brain/internal/config"
	"brain/internal/embeddings"
	"brain/internal/history"
	"brain/internal/index"
	"brain/internal/notes"
	"brain/internal/project"
	"brain/internal/search"
	"brain/internal/session"
	"brain/internal/templates"
	"brain/internal/workspace"
)

func TestFromSessionRequiresActiveSession(t *testing.T) {
	manager, _ := newTestManager(t)
	if _, err := manager.FromSession(context.Background(), 6); err == nil || !strings.Contains(err.Error(), "requires an active session") {
		t.Fatalf("expected active session error, got %v", err)
	}
}

func TestFromSessionCreatesProposalWithoutEditingTargets(t *testing.T) {
	manager, harness := newTestManager(t)
	mustInitGitRepo(t, harness.root)

	if _, err := harness.session.Start(context.Background(), session.StartRequest{
		ProjectDir: harness.root,
		Task:       "tighten session distill",
	}); err != nil {
		t.Fatalf("start session: %v", err)
	}

	if err := os.WriteFile(filepath.Join(harness.root, "main.go"), []byte("package main\nfunc main() { println(\"updated\") }\n"), 0o644); err != nil {
		t.Fatalf("write code change: %v", err)
	}
	if _, err := harness.notes.Update(".brain/context/current-state.md", notes.UpdateInput{
		Body:    stringPtr("# Current State\n\nRecorded durable context.\n"),
		Summary: "recorded durable context",
	}); err != nil {
		t.Fatalf("update durable note: %v", err)
	}
	if _, err := harness.session.RunCommand(context.Background(), session.RunRequest{
		ProjectDir:    harness.root,
		Argv:          []string{"go", "version"},
		CaptureOutput: true,
	}, nil, nil); err != nil {
		t.Fatalf("record command: %v", err)
	}

	agentsBefore, err := os.ReadFile(filepath.Join(harness.root, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read agents before: %v", err)
	}

	note, err := manager.FromSession(context.Background(), 6)
	if err != nil {
		t.Fatalf("distill session: %v", err)
	}
	if note.Type != "distill_proposal" {
		t.Fatalf("expected distill proposal note, got %q", note.Type)
	}
	if !strings.Contains(note.Content, "## Source Provenance") || !strings.Contains(note.Content, "## Proposed Updates") {
		t.Fatalf("expected provenance and targets in proposal:\n%s", note.Content)
	}
	if !strings.Contains(note.Content, "go version") || !strings.Contains(note.Content, "main.go") || !strings.Contains(note.Content, ".brain/context/current-state.md") {
		t.Fatalf("expected session-derived material in proposal:\n%s", note.Content)
	}
	if !strings.Contains(note.Content, "## Promotion Review") || !strings.Contains(note.Content, "verification_recipe [promotable]") || !strings.Contains(note.Content, "### .brain/resources/changes/tighten-session-distill.md") {
		t.Fatalf("expected promotion review and promotable target sections in proposal:\n%s", note.Content)
	}

	agentsAfter, err := os.ReadFile(filepath.Join(harness.root, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read agents after: %v", err)
	}
	if string(agentsBefore) != string(agentsAfter) {
		t.Fatalf("expected distill not to modify AGENTS.md directly")
	}
}

type testHarness struct {
	root    string
	notes   *notes.Manager
	session *session.Manager
}

func newTestManager(t *testing.T) (*Manager, testHarness) {
	t.Helper()
	root := t.TempDir()
	workspaceSvc := workspace.New(root)
	if err := workspaceSvc.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# Project Agent Contract\n\nLocal notes.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".brain", "context", "current-state.md"), []byte("---\ntitle: Current State\ntype: resource\n---\n# Current State\n\nBaseline state.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for rel, content := range map[string]string{
		".brain/context/overview.md":      "# Overview\n\nTest overview.\n",
		".brain/context/workflows.md":     "# Workflows\n\nTest workflows.\n",
		".brain/context/memory-policy.md": "# Memory Policy\n\nTest memory rules.\n",
		".brain/policy.yaml": `version: 1
project:
  name: brain
  slug: brain
  runtime: go
  memory:
    accepted_note_globs:
      - AGENTS.md
      - docs/**
      - .brain/context/**
      - .brain/planning/**
      - .brain/brainstorms/**
      - .brain/resources/**
session:
  require_task: true
  single_active: true
  active_file: .brain/session.json
  ledger_dir: .brain/sessions
preflight:
  require_brain_doctor: true
  required_docs:
    - AGENTS.md
    - .brain/context/overview.md
    - .brain/context/workflows.md
    - .brain/context/memory-policy.md
closeout:
  acceptable_history_operations:
    - update
  require_memory_update_on_repo_change: true
  verification_profiles:
    - name: tests
      commands:
        - "go test ./..."
`,
	} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	historyLog := history.New(filepath.Join(root, ".brain", "state", "history.jsonl"))
	notesManager := notes.New(workspaceSvc, templates.New(), backup.New(filepath.Join(root, ".brain", "state", "backups")), historyLog)
	projectManager := project.New(notesManager, workspaceSvc)
	store, err := index.New(filepath.Join(root, ".brain", "state", "brain.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	provider, err := embeddings.New(&config.Config{
		EmbeddingProvider: "localhash",
		EmbeddingModel:    "hash-v1",
		OutputMode:        "json",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Reindex(context.Background(), workspaceSvc, provider); err != nil {
		t.Fatal(err)
	}
	searchEngine := search.New(store, provider)
	sessionManager := session.New(historyLog)
	manager := New(notesManager, searchEngine, projectManager, historyLog, sessionManager)
	return manager, testHarness{
		root:    root,
		notes:   notesManager,
		session: sessionManager,
	}
}

func mustInitGitRepo(t *testing.T, root string) {
	t.Helper()
	runGit(t, root, "init", "-q")
	runGit(t, root, "config", "user.email", "tester@example.com")
	runGit(t, root, "config", "user.name", "tester")
	runGit(t, root, "add", ".")
	runGit(t, root, "commit", "-q", "-m", "init")
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(out))
	}
}

func stringPtr(value string) *string {
	return &value
}
