package search

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"brain/internal/embeddings"
	"brain/internal/index"
)

type Result struct {
	NotePath      string  `json:"note_path"`
	NoteTitle     string  `json:"note_title,omitempty"`
	NoteType      string  `json:"note_type,omitempty"`
	ModifiedAt    string  `json:"modified_at,omitempty"`
	Heading       string  `json:"heading"`
	Snippet       string  `json:"snippet"`
	Score         float64 `json:"score"`
	LexicalScore  float64 `json:"lexical_score,omitempty"`
	SemanticScore float64 `json:"semantic_score,omitempty"`
	RecencyBoost  float64 `json:"recency_boost,omitempty"`
	TypeBoost     float64 `json:"type_boost,omitempty"`
	ContextBoost  float64 `json:"context_boost,omitempty"`
	Source        string  `json:"source,omitempty"`
}

type Options struct {
	ActiveTask string
}

type Engine struct {
	store    *index.Store
	embedder embeddings.Provider
}

var tokenPattern = regexp.MustCompile(`[[:alnum:]_]+`)

func New(store *index.Store, embedder embeddings.Provider) *Engine {
	return &Engine{store: store, embedder: embedder}
}

func (e *Engine) Search(ctx context.Context, query string, limit int) ([]Result, error) {
	return e.SearchWithOptions(ctx, query, limit, Options{})
}

func (e *Engine) SearchWithExplain(ctx context.Context, query string, limit int) ([]Result, error) {
	return e.SearchWithExplainOptions(ctx, query, limit, Options{})
}

func (e *Engine) SearchWithOptions(ctx context.Context, query string, limit int, opts Options) ([]Result, error) {
	return e.search(ctx, query, limit, false, opts)
}

func (e *Engine) SearchWithExplainOptions(ctx context.Context, query string, limit int, opts Options) ([]Result, error) {
	return e.search(ctx, query, limit, true, opts)
}

func (e *Engine) search(ctx context.Context, query string, limit int, explain bool, opts Options) ([]Result, error) {
	fts, err := e.store.SearchFTS(ctx, query, max(limit*3, 15))
	if err != nil {
		return nil, err
	}

	combined := map[int64]*Result{}
	ftsScores := normalizeFTS(fts)
	for i, rec := range fts {
		lexicalScore := ftsScores[i] * 0.45
		combined[rec.ChunkID] = &Result{
			NotePath:   rec.NotePath,
			NoteTitle:  rec.NoteTitle,
			NoteType:   rec.NoteType,
			ModifiedAt: rec.ModifiedAt,
			Heading:    rec.Heading,
			Snippet:    rec.Snippet,
			Score:      lexicalScore,
		}
		if explain {
			combined[rec.ChunkID].LexicalScore = lexicalScore
		}
	}

	if e.embedder != nil && e.embedder.Name() != "none" {
		queryVecs, err := e.embedder.Embed(ctx, []string{query})
		if err == nil && len(queryVecs) == 1 {
			records, vectors, err := e.store.EmbeddingCandidates(ctx, e.embedder.Name(), e.embedder.Model())
			if err == nil {
				sims := make([]float64, len(vectors))
				for i, vec := range vectors {
					sims[i] = cosine(queryVecs[0], vec)
				}
				norm := normalizeDense(sims)
				for i, rec := range records {
					if norm[i] <= 0 {
						continue
					}
					semanticScore := norm[i] * 0.55
					if existing, ok := combined[rec.ChunkID]; ok {
						existing.Score += semanticScore
						if existing.Snippet == "" {
							existing.Snippet = rec.Snippet
						}
						if explain {
							existing.SemanticScore = semanticScore
						}
						continue
					}
					combined[rec.ChunkID] = &Result{
						NotePath:   rec.NotePath,
						NoteTitle:  rec.NoteTitle,
						NoteType:   rec.NoteType,
						ModifiedAt: rec.ModifiedAt,
						Heading:    rec.Heading,
						Snippet:    rec.Snippet,
						Score:      semanticScore,
					}
					if explain {
						combined[rec.ChunkID].SemanticScore = semanticScore
					}
				}
			}
		}
	}

	applyRecencyBoosts(combined, explain)
	applyTypeBoosts(combined, explain)
	applyContextBoosts(combined, opts.ActiveTask, explain)

	results := make([]Result, 0, len(combined))
	for _, result := range combined {
		if explain {
			switch {
			case result.LexicalScore > 0 && result.SemanticScore > 0:
				result.Source = "hybrid"
			case result.LexicalScore > 0:
				result.Source = "lexical"
			case result.SemanticScore > 0:
				result.Source = "semantic"
			}
		}
		results = append(results, *result)
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			if results[i].NotePath == results[j].NotePath {
				return results[i].Heading < results[j].Heading
			}
			return results[i].NotePath < results[j].NotePath
		}
		return results[i].Score > results[j].Score
	})
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func applyRecencyBoosts(combined map[int64]*Result, explain bool) {
	if len(combined) == 0 {
		return
	}
	type stampEntry struct {
		id   int64
		unix float64
	}
	var stamps []float64
	var parsed []stampEntry
	for id, result := range combined {
		if ts, err := time.Parse(time.RFC3339, result.ModifiedAt); err == nil {
			unix := float64(ts.UTC().Unix())
			parsed = append(parsed, stampEntry{id: id, unix: unix})
			stamps = append(stamps, unix)
		}
	}
	norm := normalizeDense(stamps)
	if len(norm) == 0 {
		return
	}
	for i, entry := range parsed {
		boost := norm[i] * 0.08
		combined[entry.id].Score += boost
		if explain {
			combined[entry.id].RecencyBoost = boost
		}
	}
}

func applyTypeBoosts(combined map[int64]*Result, explain bool) {
	for _, result := range combined {
		boost := noteTypeBoost(result.NoteType)
		if boost <= 0 {
			continue
		}
		result.Score += boost
		if explain {
			result.TypeBoost = boost
		}
	}
}

func applyContextBoosts(combined map[int64]*Result, activeTask string, explain bool) {
	activeTask = strings.TrimSpace(activeTask)
	if activeTask == "" {
		return
	}
	taskTokens := tokenize(activeTask)
	if len(taskTokens) == 0 {
		return
	}
	for _, result := range combined {
		candidateTokens := tokenize(strings.Join([]string{
			result.NotePath,
			result.NoteTitle,
			result.NoteType,
			result.Heading,
			result.Snippet,
		}, " "))
		if len(candidateTokens) == 0 {
			continue
		}
		matches := 0
		for token := range taskTokens {
			if _, ok := candidateTokens[token]; ok {
				matches++
			}
		}
		if matches == 0 {
			continue
		}
		boost := (float64(matches) / float64(len(taskTokens))) * 0.07
		result.Score += boost
		if explain {
			result.ContextBoost = boost
		}
	}
}

func noteTypeBoost(noteType string) float64 {
	switch strings.ToLower(strings.TrimSpace(noteType)) {
	case "decision":
		return 0.08
	case "spec":
		return 0.07
	case "change":
		return 0.06
	case "epic", "story":
		return 0.04
	case "reference", "resource":
		return 0.02
	default:
		return 0
	}
}

func tokenize(text string) map[string]struct{} {
	tokens := tokenPattern.FindAllString(strings.ToLower(text), -1)
	if len(tokens) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		out[token] = struct{}{}
	}
	return out
}

func BuildContextBlock(results []Result) string {
	if len(results) == 0 {
		return "## Relevant Context\n\n- No relevant context found.\n"
	}
	seen := map[string]struct{}{}
	lines := []string{"## Relevant Context", ""}
	for _, result := range results {
		if _, ok := seen[result.NotePath]; ok {
			continue
		}
		seen[result.NotePath] = struct{}{}
		source := fmt.Sprintf("- Source: `%s`", result.NotePath)
		if result.Heading != "" {
			source += fmt.Sprintf(" -> `%s`", result.Heading)
		}
		snippet := strings.TrimSpace(result.Snippet)
		if snippet != "" {
			source += fmt.Sprintf(": %s", snippet)
		}
		lines = append(lines, source)
		if len(seen) >= 5 {
			break
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func cosine(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		av := float64(a[i])
		bv := float64(b[i])
		dot += av * bv
		normA += av * av
		normB += bv * bv
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func normalizeFTS(records []index.ChunkRecord) []float64 {
	if len(records) == 0 {
		return nil
	}
	raw := make([]float64, len(records))
	for i, rec := range records {
		raw[i] = -rec.Score
	}
	return normalizeDense(raw)
}

func normalizeDense(values []float64) []float64 {
	if len(values) == 0 {
		return nil
	}
	minVal, maxVal := values[0], values[0]
	for _, v := range values[1:] {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}
	out := make([]float64, len(values))
	if maxVal == minVal {
		for i := range out {
			if values[i] > 0 {
				out[i] = 1
			}
		}
		return out
	}
	for i, v := range values {
		out[i] = (v - minVal) / (maxVal - minVal)
	}
	return out
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
