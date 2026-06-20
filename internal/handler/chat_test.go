package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ryanairlabs/ryta/pkg/ollama"
)

func TestChatHandlerServeHTTP(t *testing.T) {
	// Mock Ollama client that returns fixed responses
	mockClient := &mockOllamaClient{
		chatFunc: func(ctx context.Context, req ollama.ChatRequest, tokenCh chan<- string) error {
			// Simulate streaming response
			tokenCh <- "Hello"
			tokenCh <- " world!"
			return nil
		},
	}
	
	handler := NewChat(mockClient)
	
	// Create test request with valid JSON
	requestBody := `{
		"model": "test-model",
		"messages": [
			{"role": "user", "content": "Hello there"}
		]
	}`
	
	req := httptest.NewRequest("POST", "/api/chat", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	
	rr := httptest.NewRecorder()
	
	handler.ServeHTTP(rr, req)
	
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	
	// Check response headers
	contentType := rr.Result().Header.Get("Content-Type")
	if contentType != "text/event-stream" {
		t.Errorf("Expected Content-Type text/event-stream, got %s", contentType)
	}
}

func TestChatHandlerServeHTTPInvalidJSON(t *testing.T) {
	handler := NewChat(&mockOllamaClient{})
	
	req := httptest.NewRequest("POST", "/api/chat", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	
	rr := httptest.NewRecorder()
	
	handler.ServeHTTP(rr, req)
	
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", status)
	}
}

func TestChatHandlerServeHTTPMissingModel(t *testing.T) {
	handler := NewChat(&mockOllamaClient{})
	
	requestBody := `{
		"messages": [
			{"role": "user", "content": "Hello there"}
		]
	}`
	
	req := httptest.NewRequest("POST", "/api/chat", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	
	rr := httptest.NewRecorder()
	
	handler.ServeHTTP(rr, req)
	
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing model, got %d", status)
	}
}

func TestChatHandlerServeHTTPOptions(t *testing.T) {
	handler := NewChat(&mockOllamaClient{})
	
	req := httptest.NewRequest("OPTIONS", "/api/chat", nil)
	rr := httptest.NewRecorder()
	
	handler.ServeHTTP(rr, req)
	
	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf("Expected status 204 for OPTIONS, got %d", status)
	}
	
	// Check CORS headers
	expectedHeaders := map[string]string{
		"Access-Control-Allow-Origin":      "*",
		"Access-Control-Allow-Headers":     "Content-Type",
		"Access-Control-Allow-Methods":     "POST, OPTIONS",
		"Content-Type":                     "text/event-stream",
		"Cache-Control":                    "no-cache",
		"Connection":                       "keep-alive",
		"X-Accel-Buffering":                "no",
	}
	
	for header, expectedValue := range expectedHeaders {
		actualValue := rr.Result().Header.Get(header)
		if actualValue != expectedValue {
			t.Errorf("Expected header %s to be '%s', got '%s'", header, expectedValue, actualValue)
		}
	}
}

func TestChatHandlerServeHTTPMethodNotAllowed(t *testing.T) {
	handler := NewChat(&mockOllamaClient{})
	
	req := httptest.NewRequest("GET", "/api/chat", nil)
	rr := httptest.NewRecorder()
	
	handler.ServeHTTP(rr, req)
	
	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 for GET, got %d", status)
	}
}

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