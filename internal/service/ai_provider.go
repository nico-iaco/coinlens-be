package service

import (
	"context"

	"coinlens-be/internal/models"
)

// CoinAIProvider is the interface that both GeminiClient and OllamaClient implement.
// Any component that needs AI-based coin identification should depend on this interface.
type CoinAIProvider interface {
	// IdentifyCoin identifies a coin from its front and back images.
	IdentifyCoin(ctx context.Context, frontImage, backImage []byte) (*models.CoinAnalysis, error)

	// IdentifyFromSingleImage identifies a coin from a single image (e.g. reverse side for search).
	IdentifyFromSingleImage(ctx context.Context, image []byte) (*models.CoinAnalysis, error)
}

// Compile-time interface checks
var _ CoinAIProvider = (*GeminiClient)(nil)
var _ CoinAIProvider = (*OllamaClient)(nil)
