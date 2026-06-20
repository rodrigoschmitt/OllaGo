package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ryanairlabs/ryta/pkg/ollama"
)

func TestModelsHandlerServeHTTP(t *testing.T) {
	// Mock Ollama client that returns fixed models
	mockClient := &mockOllamaClient{
		modelsFunc: func(ctx context.Context) ([]ollama.ModelInfo, error) {
			return []ollama.ModelInfo{
				{Name: "test-model-1"},
				{Name: "test-model-2"},
			}, nil
		},
	}
	
	handler := NewModels(mockClient)
	
	req := httptest.NewRequest("GET", "/api/models", nil)
	rr := httptest.NewRecorder()
	
	handler.ServeHTTP(rr, req)
	
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	
	// Check content type
	contentType := rr.Result().Header.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
	
	// Parse response and validate content
	var models []ollama.ModelInfo
	if err := json.NewDecoder(rr.Body).Decode(&models); err != nil {
		t.Errorf("Error decoding JSON response: %v", err)
	}
	
	if len(models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(models))
	}
	
	if models[0].Name != "test-model-1" {
		t.Errorf("Expected first model name 'test-model-1', got '%s'", models[0].Name)
	}
}

func TestModelsHandlerServeHTTPMethodNotAllowed(t *testing.T) {
	mockClient := &mockOllamaClient{}
	handler := NewModels(mockClient)
	
	req := httptest.NewRequest("POST", "/api/models", nil)
	rr := httptest.NewRecorder()
	
	handler.ServeHTTP(rr, req)
	
	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 for POST, got %d", status)
	}
}

func TestModelsHandlerServeHTTPError(t *testing.T) {
	mockClient := &mockOllamaClient{
		modelsFunc: func(ctx context.Context) ([]ollama.ModelInfo, error) {
			return nil, &mockError{message: "connection failed"}
		},
	}
	
	handler := NewModels(mockClient)
	
	req := httptest.NewRequest("GET", "/api/models", nil)
	rr := httptest.NewRecorder()
	
	handler.ServeHTTP(rr, req)
	
	if status := rr.Code; status != http.StatusBadGateway {
		t.Errorf("Expected status 502 for error, got %d", status)
	}
}

type mockError struct {
	message string
}

func (m *mockError) Error() string {
	return m.message
}