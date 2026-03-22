package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/ryanairlabs/ryta/pkg/ollama"
)

// Models handles GET /api/models, returning the locally available Ollama models.
type Models struct {
	client *ollama.Client
}

func NewModels(client *ollama.Client) *Models {
	return &Models{client: client}
}

func (h *Models) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	models, err := h.client.Models(ctx)
	if err != nil {
		slog.Error("failed to fetch models", "err", err)
		http.Error(w, "could not reach Ollama", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}
