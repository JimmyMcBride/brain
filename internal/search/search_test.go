package search

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"brain/internal/config"
	"brain/internal/embeddings"
	"brain/internal/index"
	"brain/internal/vault"
)

func TestHybridSearchReturnsRelevantChunks(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	vaultRoot := filepath.Join(root, "vault")
	dataRoot := filepath.Join(root, "data")

	cfg := &config.Config{
		VaultPath:         vaultRoot,
		DataPath:          dataRoot,
		EmbeddingProvider: "localhash",
		EmbeddingModel:    "hash-v1",
		OutputMode:        "human",
	}
	vaultSvc := vault.New(cfg)
	if err := vaultSvc.Initialize(); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		filepath.Join(vaultRoot, "Resources", "retrieval.md"): `---
title: Retrieval Design
type: resource
---
# Hybrid Search

Blend lexical ranking with semantic similarity and rerank the merged candidates.
`,
		filepath.Join(vaultRoot, "Resources", "gardening.md"): `---
title: Garden Notes
type: resource
---
# Herbs

Basil grows well in warm sunlight with consistent watering.
`,
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	store, err := index.New(filepath.Join(dataRoot, "brain.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	provider, err := embeddings.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Reindex(ctx, vaultSvc, provider); err != nil {
		t.Fatal(err)
	}

	engine := New(store, provider)
	results, err := engine.Search(ctx, "semantic lexical retrieval", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("expected search results")
	}
	if results[0].NotePath != "Resources/retrieval.md" {
		t.Fatalf("expected retrieval note first, got %s", results[0].NotePath)
	}
	if results[0].Score <= 0 {
		t.Fatalf("expected positive score, got %f", results[0].Score)
	}
}
