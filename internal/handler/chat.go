package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/ryanairlabs/ryta/pkg/ollama"
)

type chatRequest struct {
	Model    string          `json:"model"`
	Messages []ollama.Message `json:"messages"`
}

// Chat handles POST /api/chat, proxying to Ollama and streaming tokens via SSE.
type Chat struct {
	client *ollama.Client
}

func NewChat(client *ollama.Client) *Chat {
	return &Chat{client: client}
}

func (h *Chat) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		writeCORSHeaders(w)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if req.Model == "" {
		http.Error(w, `"model" is required`, http.StatusBadRequest)
		return
	}
	if len(req.Messages) == 0 {
		http.Error(w, `"messages" must not be empty`, http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported by server", http.StatusInternalServerError)
		return
	}

	writeCORSHeaders(w)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering if present

	ctx := r.Context()
	tokenCh := make(chan string, 64)

	// Run the Ollama call in a goroutine so we can stream to the client
	// concurrently. The goroutine closes tokenCh when done.
	go func() {
		defer close(tokenCh)
		ollamaReq := ollama.ChatRequest{
			Model:    req.Model,
			Messages: req.Messages,
		}
		if err := h.client.Chat(ctx, ollamaReq, tokenCh); err != nil {
			slog.Error("ollama chat error",
				"model", req.Model,
				"err", err,
			)
			// Surface the error as an SSE event the client can detect.
			tokenCh <- fmt.Sprintf("\n\n[OllaGo error: %s]", err.Error())
		}
	}()

	for token := range tokenCh {
		select {
		case <-ctx.Done():
			// Client disconnected; stop streaming.
			slog.Info("client disconnected", "model", req.Model)
			return
		default:
		}

		encoded, _ := json.Marshal(token) // escapes special characters safely
		fmt.Fprintf(w, "data: %s\n\n", encoded)
		flusher.Flush()
	}

	// Signal end-of-stream so the client knows it's safe to re-enable input.
	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()

	slog.Info("chat complete",
		"model", req.Model,
		"turns", len(req.Messages),
	)
}

func writeCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
}
