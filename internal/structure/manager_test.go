package structure

import (
	"context"
	"os"
	"path/filepath"
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
