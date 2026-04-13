package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"coinlens-be/internal/models"

	"google.golang.org/genai"
)

type GeminiClient struct {
	client    *genai.Client
	modelName string
}

func NewGeminiClient(apiKey, modelName string) (*GeminiClient, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}
	return &GeminiClient{client: client, modelName: modelName}, nil
}

func (g *GeminiClient) IdentifyCoin(ctx context.Context, frontImage, backImage []byte, stream chan<- string) (*models.CoinAnalysis, error) {
	if stream != nil {
		defer close(stream)
	}
	if len(frontImage) == 0 || len(backImage) == 0 {
		return nil, fmt.Errorf("images cannot be empty")
	}

	prompt := "Identify this coin from the front and back images. Return ONLY a valid JSON object with the following fields: name, description, year, country, universal_id. The 'universal_id' must be a numeric identifier like Numista ID if available (return ONLY the ID value, e.g., '325206' instead of 'Numista 325206'), otherwise return an empty string. Do not include markdown code blocks."

	parts := []*genai.Part{
		genai.NewPartFromText(prompt),
		genai.NewPartFromBytes(frontImage, "image/jpeg"),
		genai.NewPartFromBytes(backImage, "image/jpeg"),
	}

	contents := []*genai.Content{
		{Parts: parts},
	}

	// 2. Call Gemini
	log.Printf("Calling Gemini API with model: %s...", g.modelName)

	var fullText string
	for resp, err := range g.client.Models.GenerateContentStream(ctx, g.modelName, contents, nil) {
		if err != nil {
			return nil, fmt.Errorf("gemini generation failed: %w", err)
		}
		if resp != nil && len(resp.Candidates) > 0 {
			for _, part := range resp.Candidates[0].Content.Parts {
				if part.Text != "" {
					fullText += part.Text
					if stream != nil {
						stream <- part.Text
					}
				}
			}
		}
	}

	// Cleanup markdown if present (e.g. ```json ... ```)
	fullText = strings.TrimPrefix(fullText, "```json")
	fullText = strings.TrimPrefix(fullText, "```")
	fullText = strings.TrimSuffix(fullText, "```")
	fullText = strings.TrimSpace(fullText)

	var result models.CoinAnalysis
	if err := json.Unmarshal([]byte(fullText), &result); err != nil {
		return nil, fmt.Errorf("failed to parse gemini response: %w, text: %s", err, fullText)
	}
	log.Printf("Successfully parsed Gemini response")

	return &result, nil
}

func (g *GeminiClient) IdentifyFromSingleImage(ctx context.Context, image []byte, stream chan<- string) (*models.CoinAnalysis, error) {
	if stream != nil {
		defer close(stream)
	}
	if len(image) == 0 {
		return nil, fmt.Errorf("image cannot be empty")
	}

	prompt := "Identify this coin from the image. Return ONLY a valid JSON object with the following fields: name, description, year, country, universal_id. The 'universal_id' must be a numeric identifier like Numista ID if available (return ONLY the ID value, e.g., '325206' instead of 'Numista 325206'), otherwise return an empty string. Do not include markdown code blocks."

	parts := []*genai.Part{
		genai.NewPartFromText(prompt),
		genai.NewPartFromBytes(image, "image/jpeg"),
	}

	contents := []*genai.Content{
		{Parts: parts},
	}

	log.Printf("Calling Gemini API for single image identification...")
	var fullText string
	for resp, err := range g.client.Models.GenerateContentStream(ctx, g.modelName, contents, nil) {
		if err != nil {
			return nil, fmt.Errorf("gemini generation failed: %w", err)
		}
		if resp != nil && len(resp.Candidates) > 0 {
			for _, part := range resp.Candidates[0].Content.Parts {
				if part.Text != "" {
					fullText += part.Text
					if stream != nil {
						stream <- part.Text
					}
				}
			}
		}
	}

	fullText = strings.TrimPrefix(fullText, "```json")
	fullText = strings.TrimPrefix(fullText, "```")
	fullText = strings.TrimSuffix(fullText, "```")
	fullText = strings.TrimSpace(fullText)

	var result models.CoinAnalysis
	if err := json.Unmarshal([]byte(fullText), &result); err != nil {
		return nil, fmt.Errorf("failed to parse gemini response: %w, text: %s", err, fullText)
	}

	return &result, nil
}
