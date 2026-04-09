package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type OpenAIProvider struct {
	model  string
	apiKey string
	client *http.Client
}

type openAIEmbeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type openAIEmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}

func NewOpenAIProvider(model string) (*OpenAIProvider, error) {
	if model == "" {
		model = "text-embedding-3-small"
	}
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is not set")
	}
	return &OpenAIProvider{
		model:  model,
		apiKey: key,
		client: &http.Client{Timeout: 45 * time.Second},
	}, nil
}

func (p *OpenAIProvider) Name() string  { return "openai" }
func (p *OpenAIProvider) Model() string { return p.model }

func (p *OpenAIProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	reqBody, err := json.Marshal(openAIEmbeddingRequest{
		Input: texts,
		Model: p.model,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/embeddings", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai embeddings request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openai embeddings returned %s", resp.Status)
	}
	var payload openAIEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode openai response: %w", err)
	}
	out := make([][]float32, len(payload.Data))
	for _, item := range payload.Data {
		out[item.Index] = item.Embedding
	}
	return out, nil
}
