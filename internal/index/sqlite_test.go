package index

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"brain/internal/config"
	"brain/internal/embeddings"
	"brain/internal/vault"
)

func TestReindexBuildsStatsAndSupportsSanitizedFTS(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	vaultRoot := filepath.Join(root, "vault")
	dataRoot := filepath.Join(root, "data")

	cfg := &config.Config{
		VaultPath:         vaultRoot,
		DataPath:          dataRoot,
		EmbeddingProvider: "localhash",
		EmbeddingModel:    "hash-v1",
		OutputMode:        "json",
	}
	vaultSvc := vault.New(cfg)
	if err := vaultSvc.Initialize(); err != nil {
		t.Fatal(err)
	}

	projectPath := filepath.Join(vaultRoot, "Projects", "alpha.md")
	resourcePath := filepath.Join(vaultRoot, "Resources", "network.md")
	if err := os.WriteFile(projectPath, []byte(`---
title: Alpha Project
type: project
---
# Plan

Hybrid retrieval keeps lexical relevance and semantic recall balanced.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(resourcePath, []byte(`---
title: Network Notes
type: resource
topic: networking
---
# Latency

Local-first tools benefit from fast indexing.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := New(filepath.Join(dataRoot, "brain.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	provider, err := embeddings.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	stats, err := store.Reindex(ctx, vaultSvc, provider)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Notes != 2 {
		t.Fatalf("expected 2 notes, got %d", stats.Notes)
	}
	if stats.Chunks < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", stats.Chunks)
	}
	if stats.Embeddings != stats.Chunks {
		t.Fatalf("expected embeddings to match chunks, got %d vs %d", stats.Embeddings, stats.Chunks)
	}

	dbStats, err := store.Stats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if dbStats != stats {
		t.Fatalf("stats mismatch: %+v vs %+v", dbStats, stats)
	}

	results, err := store.SearchFTS(ctx, "## hybrid: retrieval!", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("expected sanitized FTS query to return results")
	}
	if results[0].NotePath != "Projects/alpha.md" {
		t.Fatalf("expected Projects/alpha.md first, got %s", results[0].NotePath)
	}
}
