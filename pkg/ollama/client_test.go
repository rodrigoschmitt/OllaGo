package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientModels(t *testing.T) {
	// Create a test server that returns mock models
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/api/tags" {
			t.Errorf("Expected /api/tags path, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(struct {
			Models []ModelInfo `json:"models"`
		}{
			Models: []ModelInfo{
				{Name: "test-model-1"},
				{Name: "test-model-2"},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	models, err := client.Models(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(models))
	}

	if models[0].Name != "test-model-1" {
		t.Errorf("Expected first model name 'test-model-1', got '%s'", models[0].Name)
	}
}

func TestClientChat(t *testing.T) {
	// Create a test server that returns mock chat responses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/api/chat" {
			t.Errorf("Expected /api/chat path, got %s", r.URL.Path)
		}

		// Verify the request body contains expected fields
		var req ChatRequest
		body := make([]byte, 1024) // Buffer to read request body
		n, _ := r.Body.Read(body)
		body = body[:n]
		
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("Failed to unmarshal request: %v", err)
		}
		
		if req.Model != "test-model" {
			t.Errorf("Expected model 'test-model', got '%s'", req.Model)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		
		// Send mock response tokens
		response1 := `{"message": {"role": "assistant", "content": "Hello"}, "done": false}` + "\n"
		response2 := `{"message": {"role": "assistant", "content": " world!"}, "done": true}` + "\n"
		
		w.Write([]byte(response1))
		w.Write([]byte(response2))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()
	
	// Mock token channel
	tokenCh := make(chan string, 10)

	go func() {
		defer close(tokenCh)
		req := ChatRequest{
			Model: "test-model",
			Messages: []Message{
				{Role: "user", Content: "Hello"},
			},
		}
		
		err := client.Chat(ctx, req, tokenCh)
		if err != nil {
			t.Errorf("Expected no error from Chat, got %v", err)
		}
	}()

	// Collect tokens
	var tokens []string
	for token := range tokenCh {
		tokens = append(tokens, token)
	}

	if len(tokens) != 2 {
		t.Errorf("Expected 2 tokens, got %d", len(tokens))
	}

	if tokens[0] != "Hello" {
		t.Errorf("Expected first token 'Hello', got '%s'", tokens[0])
	}
	
	if tokens[1] != " world!" {
		t.Errorf("Expected second token ' world!', got '%s'", tokens[1])
	}
}

func TestClientModelsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	_, err := client.Models(ctx)
	if err == nil {
		t.Error("Expected error when calling Models with bad status")
	}
}

func TestClientChatError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()
	tokenCh := make(chan string, 10)

	go func() {
		defer close(tokenCh)
		req := ChatRequest{
			Model: "test-model",
			Messages: []Message{
				{Role: "user", Content: "Hello"},
			},
		}
		
		err := client.Chat(ctx, req, tokenCh)
		if err == nil {
			t.Error("Expected error when calling Chat with bad status")
		}
	}()
	
	// Wait for async operation to complete
	close(tokenCh)
}