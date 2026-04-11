package search

import (
	"context"
	"math"
	"sort"

	"brain/internal/embeddings"
	"brain/internal/index"
)

type Result struct {
	NotePath      string  `json:"note_path"`
	Heading       string  `json:"heading"`
	Snippet       string  `json:"snippet"`
	Score         float64 `json:"score"`
	LexicalScore  float64 `json:"lexical_score,omitempty"`
	SemanticScore float64 `json:"semantic_score,omitempty"`
	Source        string  `json:"source,omitempty"`
}

type Engine struct {
	store    *index.Store
	embedder embeddings.Provider
}

func New(store *index.Store, embedder embeddings.Provider) *Engine {
	return &Engine{store: store, embedder: embedder}
}

func (e *Engine) Search(ctx context.Context, query string, limit int) ([]Result, error) {
	return e.search(ctx, query, limit, false)
}

func (e *Engine) SearchWithExplain(ctx context.Context, query string, limit int) ([]Result, error) {
	return e.search(ctx, query, limit, true)
}

func (e *Engine) search(ctx context.Context, query string, limit int, explain bool) ([]Result, error) {
	fts, err := e.store.SearchFTS(ctx, query, max(limit*3, 15))
	if err != nil {
		return nil, err
	}

	combined := map[int64]*Result{}
	ftsScores := normalizeFTS(fts)
	for i, rec := range fts {
		lexicalScore := ftsScores[i] * 0.45
		combined[rec.ChunkID] = &Result{
			NotePath: rec.NotePath,
			Heading:  rec.Heading,
			Snippet:  rec.Snippet,
			Score:    lexicalScore,
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
						NotePath: rec.NotePath,
						Heading:  rec.Heading,
						Snippet:  rec.Snippet,
						Score:    semanticScore,
					}
					if explain {
						combined[rec.ChunkID].SemanticScore = semanticScore
					}
				}
			}
		}
	}

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
