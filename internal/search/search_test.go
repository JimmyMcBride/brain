package search

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"brain/internal/config"
	"brain/internal/embeddings"
	"brain/internal/index"
	"brain/internal/workspace"
)

func TestHybridSearchReturnsRelevantChunks(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	cfg := &config.Config{
		EmbeddingProvider: "localhash",
		EmbeddingModel:    "hash-v1",
		OutputMode:        "human",
	}
	workspaceSvc := workspace.New(root)
	if err := workspaceSvc.Initialize(); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		filepath.Join(root, "AGENTS.md"): `# Retrieval Design

Blend lexical ranking with semantic similarity and rerank the merged candidates.
`,
		filepath.Join(root, "docs", "gardening.md"): `# Garden Notes

Basil grows well in warm sunlight with consistent watering.
`,
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	store, err := index.New(filepath.Join(root, ".brain", "state", "brain.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	provider, err := embeddings.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Reindex(ctx, workspaceSvc, provider); err != nil {
		t.Fatal(err)
	}
	engine := New(store, provider)
	results, err := engine.Search(ctx, "semantic lexical retrieval", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 || results[0].NotePath != "AGENTS.md" || results[0].Score <= 0 {
		t.Fatalf("unexpected search results: %+v", results)
	}
}

func TestSearchWithExplainIncludesContributionBreakdown(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	cfg := &config.Config{
		EmbeddingProvider: "localhash",
		EmbeddingModel:    "hash-v1",
		OutputMode:        "human",
	}
	workspaceSvc := workspace.New(root)
	if err := workspaceSvc.Initialize(); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		filepath.Join(root, "AGENTS.md"): `# Retrieval Design

Blend lexical ranking with semantic similarity and rerank the merged candidates.
`,
		filepath.Join(root, "docs", "gardening.md"): `# Garden Notes

Basil grows well in warm sunlight with consistent watering.
`,
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	store, err := index.New(filepath.Join(root, ".brain", "state", "brain.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	provider, err := embeddings.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Reindex(ctx, workspaceSvc, provider); err != nil {
		t.Fatal(err)
	}

	engine := New(store, provider)
	results, err := engine.SearchWithExplain(ctx, "semantic lexical retrieval", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("expected explain results")
	}
	first := results[0]
	if first.NotePath != "AGENTS.md" || first.Score <= 0 {
		t.Fatalf("unexpected search results: %+v", results)
	}
	if first.Source != "hybrid" {
		t.Fatalf("expected hybrid classification, got %+v", first)
	}
	if first.LexicalScore <= 0 || first.SemanticScore <= 0 {
		t.Fatalf("expected both score contributions, got %+v", first)
	}
}
