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

	geminiClient, err := service.NewGeminiClient(cfg.GeminiAPIKey)
	if err != nil {
		log.Fatalf("Could not create Gemini client: %v", err)
	}

	storageService := service.NewStorageService("uploads")

	coinHandler := handler.NewCoinHandler(db, geminiClient, storageService)

	// Create a multiplexer
	mux := http.NewServeMux()
	mux.HandleFunc("/api/coins/identify", coinHandler.IdentifyCoin)
	mux.HandleFunc("/api/coins", coinHandler.GetCoins)

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
