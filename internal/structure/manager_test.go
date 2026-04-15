package structure

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"brain/internal/index"
	"brain/internal/workspace"
)

func TestFreshnessReportsMissingStaleAndFresh(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	workspaceSvc := workspace.New(root)
	if err := workspaceSvc.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := index.New(filepath.Join(root, ".brain", "state", "brain.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	manager, err := New(store, workspaceSvc)
	if err != nil {
		t.Fatal(err)
	}

	status, err := manager.Freshness(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status.State != "missing" || status.Reason != "structure metadata missing" {
		t.Fatalf("expected missing structural state, got %+v", status)
	}

	manifest, err := manager.BuildManifest()
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.WriteState(ctx, State{
		IndexedAt:          NowUTCString(),
		WorkspaceSignature: manifest.Signature,
		IndexedFileCount:   manifest.FileCount,
		ItemCount:          0,
		BoundaryCount:      0,
		EntrypointCount:    0,
		ConfigSurfaceCount: 0,
		TestSurfaceCount:   0,
	}); err != nil {
		t.Fatal(err)
	}
	status, err = manager.Freshness(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status.State != "fresh" || status.Reason != "workspace matches" {
		t.Fatalf("expected fresh structural state, got %+v", status)
	}

	next := time.Now().Add(2 * time.Second)
	if err := os.MkdirAll(filepath.Join(root, "internal", "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "app", "app.go"), []byte("package app\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(filepath.Join(root, "internal", "app", "app.go"), next, next); err != nil {
		t.Fatal(err)
	}
	status, err = manager.Freshness(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status.State != "stale" || status.Reason != "workspace signature changed" {
		t.Fatalf("expected stale structural state after repo change, got %+v", status)
	}
}

func TestManifestIgnoresRuntimeAndHeavyDirectories(t *testing.T) {
	root := t.TempDir()
	workspaceSvc := workspace.New(root)
	if err := workspaceSvc.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "node_modules", "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "node_modules", "pkg", "index.js"), []byte("console.log('x')\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".brain", "state", "scratch.txt"), []byte("scratch"), 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := index.New(filepath.Join(root, ".brain", "state", "brain.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	manager, err := New(store, workspaceSvc)
	if err != nil {
		t.Fatal(err)
	}

	manifest, err := manager.BuildManifest()
	if err != nil {
		t.Fatal(err)
	}
	if manifest.FileCount != 1 {
		t.Fatalf("expected manifest to ignore runtime/heavy dirs, got %+v", manifest)
	}
}

func TestRebuildAndSnapshotReturnGroupedStructure(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	workspaceSvc := workspace.New(root)
	if err := workspaceSvc.Initialize(); err != nil {
		t.Fatal(err)
	}
	for path, body := range map[string]string{
		"go.mod":                         "module example.com/test\n\ngo 1.26\n",
		"cmd/brain/main.go":              "package main\nfunc main() {}\n",
		"internal/search/search.go":      "package search\n",
		"internal/search/search_test.go": "package search\n",
		".github/workflows/ci.yml":       "name: ci\n",
		"config/app.yaml":                "name: app\n",
	} {
		if err := os.MkdirAll(filepath.Join(root, filepath.Dir(path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, path), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	store, err := index.New(filepath.Join(root, ".brain", "state", "brain.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	manager, err := New(store, workspaceSvc)
	if err != nil {
		t.Fatal(err)
	}

	snapshot, err := manager.Rebuild(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Summary.Runtime != "go" {
		t.Fatalf("expected go runtime, got %#v", snapshot.Summary)
	}
	if len(snapshot.Boundaries) == 0 || len(snapshot.Entrypoints) == 0 || len(snapshot.ConfigSurfaces) == 0 || len(snapshot.TestSurfaces) == 0 {
		t.Fatalf("expected grouped structural items, got %#v", snapshot)
	}

	filtered, err := manager.Snapshot(ctx, "internal/search")
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered.Boundaries) == 0 || filtered.Boundaries[0].Path != "internal/search/" {
		t.Fatalf("expected filtered boundary under internal/search, got %#v", filtered.Boundaries)
	}
	for _, item := range append(append(filtered.Entrypoints, filtered.ConfigSurfaces...), filtered.TestSurfaces...) {
		if !strings.HasPrefix(item.Path, "internal/search") {
			t.Fatalf("expected filtered snapshot paths under internal/search, got %#v", filtered)
		}
	}
}

func TestBoundaryGraphBuildsCompilerFacingRelations(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	workspaceSvc := workspace.New(root)
	if err := workspaceSvc.Initialize(); err != nil {
		t.Fatal(err)
	}
	for path, body := range map[string]string{
		"go.mod":                               "module example.com/test\n\ngo 1.26\n",
		"cmd/brain/main.go":                    "package main\nfunc main() {}\n",
		"internal/search/search.go":            "package search\n",
		"internal/search/search_test.go":       "package search\n",
		"internal/session/manager.go":          "package session\n",
		"internal/session/manager_test.go":     "package session\n",
		"docs/usage.md":                        "# usage\n",
		".github/workflows/ci.yml":             "name: ci\n",
		"config/app.yaml":                      "name: app\n",
		"scripts/refresh-global-brain.sh":      "#!/usr/bin/env sh\n",
		".brain/context/current-state.md":      "# Current State\n",
		".brain/context/current-state_test.go": "package context\n",
	} {
		if err := os.MkdirAll(filepath.Join(root, filepath.Dir(path)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, path), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	store, err := index.New(filepath.Join(root, ".brain", "state", "brain.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	manager, err := New(store, workspaceSvc)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := manager.Rebuild(ctx); err != nil {
		t.Fatal(err)
	}
	graph, err := manager.BoundaryGraph(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Boundaries) == 0 {
		t.Fatal("expected boundary graph records")
	}

	searchBoundary := graph.BoundaryByID("internal/search")
	if searchBoundary == nil {
		t.Fatalf("expected internal/search boundary in graph: %#v", graph.Boundaries)
	}
	if !contains(searchBoundary.Files, "internal/search/search.go") {
		t.Fatalf("expected search source file mapping: %#v", searchBoundary)
	}
	if !contains(searchBoundary.OwnedTests, "internal/search/search_test.go") {
		t.Fatalf("expected search test ownership: %#v", searchBoundary)
	}
	if !contains(searchBoundary.AdjacentBoundaries, "internal/session") {
		t.Fatalf("expected sibling boundary adjacency: %#v", searchBoundary)
	}
	if len(searchBoundary.Responsibilities) == 0 {
		t.Fatalf("expected derived responsibilities: %#v", searchBoundary)
	}

	if got := graph.BoundaryForFile("internal/search/search.go"); got == nil || got.ID != "internal/search" {
		t.Fatalf("expected deepest file-to-boundary mapping, got %#v", got)
	}
	if got := graph.BoundaryForFile("cmd/brain/main.go"); got == nil || got.ID != "cmd/brain" {
		t.Fatalf("expected nested command boundary mapping, got %#v", got)
	}
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
