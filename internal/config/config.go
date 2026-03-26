package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL   string
	Port          string
	GeminiAPIKey  string
	GeminiModel   string
	AIProvider    string // "gemini" (default) or "ollama"
	OllamaBaseURL string // e.g. http://localhost:11434
	OllamaModel   string // e.g. llava, llama3.2-vision, gemma3
}

func LoadConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	return &Config{
		DatabaseURL:   getEnv("DATABASE_URL", ""),
		Port:          getEnv("PORT", "8080"),
		GeminiAPIKey:  getEnv("GEMINI_API_KEY", ""),
		GeminiModel:   getEnv("GEMINI_MODEL", "gemini-3-flash-preview"),
		AIProvider:    getEnv("AI_PROVIDER", "gemini"),
		OllamaBaseURL: getEnv("OLLAMA_BASE_URL", "http://localhost:11434"),
		OllamaModel:   getEnv("OLLAMA_MODEL", "gemma3"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
