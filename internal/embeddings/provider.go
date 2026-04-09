package embeddings

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"strings"

	"brain/internal/config"
)

type Provider interface {
	Name() string
	Model() string
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

type NoopProvider struct{}

func (NoopProvider) Name() string  { return "none" }
func (NoopProvider) Model() string { return "none" }
func (NoopProvider) Embed(context.Context, []string) ([][]float32, error) {
	return nil, nil
}

type LocalHashProvider struct {
	model string
	dims  int
}

func New(cfg *config.Config) (Provider, error) {
	switch strings.ToLower(cfg.EmbeddingProvider) {
	case "", "localhash", "local":
		return &LocalHashProvider{model: cfg.EmbeddingModel, dims: 256}, nil
	case "openai":
		return NewOpenAIProvider(cfg.EmbeddingModel)
	case "none":
		return NoopProvider{}, nil
	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s", cfg.EmbeddingProvider)
	}
}

func (p *LocalHashProvider) Name() string  { return "localhash" }
func (p *LocalHashProvider) Model() string { return p.model }

func (p *LocalHashProvider) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, 0, len(texts))
	for _, text := range texts {
		vec := make([]float32, p.dims)
		for _, token := range tokenize(text) {
			h := fnv.New64a()
			_, _ = h.Write([]byte(token))
			idx := int(h.Sum64() % uint64(p.dims))
			vec[idx] += 1
		}
		normalize(vec)
		out = append(out, vec)
	}
	return out, nil
}

func tokenize(text string) []string {
	return strings.Fields(strings.ToLower(strings.NewReplacer(
		"\n", " ",
		"\t", " ",
		".", " ",
		",", " ",
		";", " ",
		":", " ",
		"(", " ",
		")", " ",
		"[", " ",
		"]", " ",
	).Replace(text)))
}

func normalize(vec []float32) {
	var sum float64
	for _, v := range vec {
		sum += float64(v * v)
	}
	if sum == 0 {
		return
	}
	norm := float32(math.Sqrt(sum))
	for i := range vec {
		vec[i] /= norm
	}
}
