package embeddings

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewOpenAIProviderRequiresKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	if _, err := NewOpenAIProvider(""); err == nil {
		t.Fatal("expected missing API key error")
	}
}

func TestOpenAIProviderEmbedSuccess(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	var seenAuth string
	var seenModel string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		var req openAIEmbeddingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		seenModel = req.Model
		_ = json.NewEncoder(w).Encode(openAIEmbeddingResponse{
			Data: []struct {
				Embedding []float32 `json:"embedding"`
				Index     int       `json:"index"`
			}{
				{Embedding: []float32{0.1, 0.2}, Index: 0},
				{Embedding: []float32{0.3, 0.4}, Index: 1},
			},
		})
	}))
	defer server.Close()

	provider, err := NewOpenAIProvider("text-embedding-3-small")
	if err != nil {
		t.Fatal(err)
	}
	provider.baseURL = server.URL
	provider.client = server.Client()

	vectors, err := provider.Embed(context.Background(), []string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	if seenAuth != "Bearer test-key" {
		t.Fatalf("unexpected auth header: %s", seenAuth)
	}
	if seenModel != "text-embedding-3-small" {
		t.Fatalf("unexpected model: %s", seenModel)
	}
	if len(vectors) != 2 || len(vectors[1]) != 2 {
		t.Fatalf("unexpected vectors: %+v", vectors)
	}
}

func TestOpenAIProviderEmbedNon2xx(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	provider, err := NewOpenAIProvider("")
	if err != nil {
		t.Fatal(err)
	}
	provider.baseURL = server.URL
	provider.client = server.Client()

	if _, err := provider.Embed(context.Background(), []string{"a"}); err == nil {
		t.Fatal("expected non-2xx error")
	}
}
