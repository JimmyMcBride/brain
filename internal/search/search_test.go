package search

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

func TestSearchRecencyBoostPrefersNewerNotes(t *testing.T) {
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
		filepath.Join(root, "docs", "older.md"): `---
title: Older Retrieval Note
type: resource
updated: 2026-04-10T00:00:00Z
---
# Retrieval

Local retrieval needs observability and ranking.
`,
		filepath.Join(root, "docs", "newer.md"): `---
title: Newer Retrieval Note
type: resource
updated: 2026-04-12T00:00:00Z
---
# Retrieval

Local retrieval needs observability and ranking.
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
	results, err := engine.SearchWithExplain(ctx, "retrieval observability ranking", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %+v", results)
	}
	if results[0].NotePath != "docs/newer.md" {
		t.Fatalf("expected newer note first, got %+v", results)
	}
	if results[0].RecencyBoost <= results[1].RecencyBoost {
		t.Fatalf("expected newer note to get higher recency boost, got %+v", results[:2])
	}
}

func TestSearchTypeBoostPrefersDecisionNotes(t *testing.T) {
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
		filepath.Join(root, ".brain", "resources", "decisions", "search-strategy.md"): `---
title: Search Strategy
type: decision
updated: 2026-04-12T00:00:00Z
---
# Search Strategy

Retrieval ranking should prefer stable project memory.
`,
		filepath.Join(root, ".brain", "resources", "references", "search-strategy.md"): `---
title: Search Strategy Reference
type: resource
updated: 2026-04-12T00:00:00Z
---
# Search Strategy

Retrieval ranking should prefer stable project memory.
`,
	}
	for path, content := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
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
	results, err := engine.SearchWithExplain(ctx, "retrieval ranking stable memory", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %+v", results)
	}
	if results[0].NoteType != "decision" {
		t.Fatalf("expected decision note first, got %+v", results)
	}
	if results[0].TypeBoost <= results[1].TypeBoost {
		t.Fatalf("expected decision note to get higher type boost, got %+v", results[:2])
	}
}

func TestSearchActiveTaskBoostPrefersMatchingWorkContext(t *testing.T) {
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
		filepath.Join(root, "docs", "session-distill.md"): `---
title: Workflow Notes
type: doc
updated: 2026-04-12T00:00:00Z
---
# Workflow

The workflow should stay explicit and deterministic.
`,
		filepath.Join(root, "docs", "release-workflow.md"): `---
title: Workflow Notes
type: doc
updated: 2026-04-12T00:00:00Z
---
# Workflow

The workflow should stay explicit and deterministic.
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
	withContext, err := engine.SearchWithExplainOptions(ctx, "workflow explicit deterministic", 5, Options{ActiveTask: "implement session distill memory updates"})
	if err != nil {
		t.Fatal(err)
	}
	if len(withContext) < 2 {
		t.Fatalf("expected at least 2 results, got %+v", withContext)
	}
	if withContext[0].NotePath != "docs/session-distill.md" {
		t.Fatalf("expected active-task-matching note first, got %+v", withContext)
	}
	if withContext[0].ContextBoost <= 0 {
		t.Fatalf("expected context boost on matching note, got %+v", withContext[0])
	}

	withoutContext, err := engine.Search(ctx, "workflow", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(withoutContext) < 2 {
		t.Fatalf("expected at least 2 baseline results, got %+v", withoutContext)
	}
	if strings.TrimSpace(withoutContext[0].NotePath) == "" {
		t.Fatalf("expected note path in baseline results, got %+v", withoutContext)
	}
}

func TestBuildContextBlockDedupesNotes(t *testing.T) {
	block := BuildContextBlock([]Result{
		{NotePath: "docs/session.md", Heading: "Overview", Snippet: "Session guidance."},
		{NotePath: "docs/session.md", Heading: "Details", Snippet: "More details."},
		{NotePath: "AGENTS.md", Heading: "", Snippet: "Project contract."},
	})

	if strings.Count(block, "docs/session.md") != 1 {
		t.Fatalf("expected deduped session note in context block:\n%s", block)
	}
	if !strings.Contains(block, "## Relevant Context") || !strings.Contains(block, "AGENTS.md") {
		t.Fatalf("unexpected context block contents:\n%s", block)
	}
}
