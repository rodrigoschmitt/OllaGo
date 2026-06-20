package handler

import (
	"context"

	"github.com/ryanairlabs/ryta/pkg/ollama"
)

// Mock client struct that implements ollama.Client interface
type mockOllamaClient struct {
	chatFunc func(ctx context.Context, req ollama.ChatRequest, tokenCh chan<- string) error
	modelsFunc func(ctx context.Context) ([]ollama.ModelInfo, error)
}

func (m *mockOllamaClient) Chat(ctx context.Context, req ollama.ChatRequest, tokenCh chan<- string) error {
	if m.chatFunc != nil {
		return m.chatFunc(ctx, req, tokenCh)
	}
	return nil
}

func (m *mockOllamaClient) Models(ctx context.Context) ([]ollama.ModelInfo, error) {
	if m.modelsFunc != nil {
		return m.modelsFunc(ctx)
	}
	return []ollama.ModelInfo{{Name: "test-model"}}, nil
}