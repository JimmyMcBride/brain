package index

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"brain/internal/config"
	"brain/internal/embeddings"
	"brain/internal/workspace"
)

func TestReindexBuildsStatsAndSupportsSanitizedFTS(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	cfg := &config.Config{
		EmbeddingProvider: "localhash",
		EmbeddingModel:    "hash-v1",
		OutputMode:        "json",
	}
	workspaceSvc := workspace.New(root)
	if err := workspaceSvc.Initialize(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# Contract\n\nHybrid retrieval keeps lexical relevance and semantic recall balanced.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "network.md"), []byte(`# Network Notes

Local-first tools benefit from fast indexing.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	store, err := New(filepath.Join(root, ".brain", "state", "brain.sqlite3"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	provider, err := embeddings.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	stats, err := store.Reindex(ctx, workspaceSvc, provider)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Notes != 2 || stats.Chunks < 2 || stats.Embeddings != stats.Chunks {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	results, err := store.SearchFTS(ctx, "## hybrid: retrieval!", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 || results[0].NotePath != "AGENTS.md" {
		t.Fatalf("unexpected results: %+v", results)
	}
}
