package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ryanairlabs/ryta/internal/handler"
	"github.com/ryanairlabs/ryta/pkg/ollama"
)

func main() {
	// Structured JSON logging — easy to pipe into any log aggregator.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	ollamaURL := env("OLLAMA_URL", "http://localhost:11434")
	addr := env("ADDR", ":8080")

	ollamaClient := ollama.NewClient(ollamaURL)
	chatHandler := handler.NewChat(ollamaClient)
	modelsHandler := handler.NewModels(ollamaClient)

	mux := http.NewServeMux()
	mux.Handle("/api/chat", chatHandler)
	mux.Handle("/api/models", modelsHandler)
	// Serve the frontend from the ./static directory.
	mux.Handle("/", http.FileServer(http.Dir("static")))

	srv := &http.Server{
		Addr:        addr,
		Handler:     mux,
		ReadTimeout: 15 * time.Second,
		// WriteTimeout must be 0 (unlimited) for SSE streaming connections.
		WriteTimeout: 0,
		IdleTimeout:  60 * time.Second,
	}

	// Start the server in the background.
	go func() {
		slog.Info("OllaGo server started",
			"addr", addr,
			"ollama", ollamaURL,
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()

	// Block until SIGINT or SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down — draining active connections...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "err", err)
		os.Exit(1)
	}
	slog.Info("server stopped cleanly")
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
