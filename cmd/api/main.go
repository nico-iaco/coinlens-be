package main

import (
	"fmt"
	"log"
	"net/http"

	"coinlens-be/internal/config"
	"coinlens-be/internal/database"
	"coinlens-be/internal/handler"
	"coinlens-be/internal/middleware"
	"coinlens-be/internal/service"
)

func main() {
	fmt.Println("Starting CoinLens Backend...")

	cfg := config.LoadConfig()

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Could not connect to database: %v", err)
	}
	defer db.Close()

	// Select AI provider based on configuration
	var aiProvider service.CoinAIProvider
	switch cfg.AIProvider {
	case "ollama":
		log.Printf("Using Ollama AI provider (model: %s, url: %s)", cfg.OllamaModel, cfg.OllamaBaseURL)
		aiProvider = service.NewOllamaClient(cfg.OllamaBaseURL, cfg.OllamaModel)
	default:
		log.Printf("Using Gemini AI provider (model: %s)", cfg.GeminiModel)
		geminiClient, err := service.NewGeminiClient(cfg.GeminiAPIKey, cfg.GeminiModel)
		if err != nil {
			log.Fatalf("Could not create Gemini client: %v", err)
		}
		aiProvider = geminiClient
	}

	storageService := service.NewStorageService("uploads")

	coinHandler := handler.NewCoinHandler(db, aiProvider, storageService)

	// Create a multiplexer
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/coins/identify", coinHandler.IdentifyCoin)
	mux.HandleFunc("POST /api/coins/search", coinHandler.SearchSimilarCoins)
	mux.HandleFunc("GET /api/coins", coinHandler.GetCoins)
	mux.HandleFunc("POST /api/coins", coinHandler.CreateCoin)
	mux.HandleFunc("PUT /api/coins/{id}", coinHandler.UpdateCoin)
	mux.HandleFunc("DELETE /api/coins/{id}", coinHandler.DeleteCoin)
	mux.HandleFunc("POST /api/coins/{id}/identify", coinHandler.ReidentifyCoin)

	// Serve static files from "uploads" directory
	fs := http.FileServer(http.Dir("uploads"))
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", fs))

	log.Printf("Server running on port %s", cfg.Port)

	// Wrap the mux with the logging middleware
	handlerWithLogging := middleware.LoggingMiddleware(mux)

	if err := http.ListenAndServe(":"+cfg.Port, handlerWithLogging); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
