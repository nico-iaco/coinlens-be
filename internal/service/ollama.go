package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"coinlens-be/internal/models"
)

// OllamaClient calls a local Ollama instance for coin identification.
// It uses the /api/chat endpoint with vision-capable models (e.g. llava, llama3.2-vision, gemma3).
type OllamaClient struct {
	baseURL    string
	modelName  string
	httpClient *http.Client
}

// NewOllamaClient creates a new OllamaClient.
func NewOllamaClient(baseURL, modelName string) *OllamaClient {
	return &OllamaClient{
		baseURL:   strings.TrimRight(baseURL, "/"),
		modelName: modelName,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // LLMs can be slow locally
		},
	}
}

// ollamaMessage is a single message in the Ollama chat request.
type ollamaMessage struct {
	Role    string   `json:"role"`
	Content string   `json:"content"`
	Images  []string `json:"images,omitempty"` // base64-encoded images
}

// ollamaChatRequest is the body sent to POST /api/chat.
type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

// ollamaChatResponse is the response from POST /api/chat.
type ollamaChatResponse struct {
	Message ollamaMessage `json:"message"`
}

// IdentifyCoin identifies a coin from front and back images.
func (o *OllamaClient) IdentifyCoin(ctx context.Context, frontImage, backImage []byte, stream chan<- string) (*models.CoinAnalysis, error) {
	if len(frontImage) == 0 || len(backImage) == 0 {
		if stream != nil {
			close(stream)
		}
		return nil, fmt.Errorf("images cannot be empty")
	}

	prompt := "Identify this coin from the front and back images. Return ONLY a valid JSON object with the following fields: name, description, year, country, universal_id. The 'universal_id' must be a numeric identifier like Numista ID if available (return ONLY the ID value, e.g., '325206' instead of 'Numista 325206'), otherwise return an empty string. Do not include markdown code blocks or any text outside the JSON object."

	images := []string{
		base64.StdEncoding.EncodeToString(frontImage),
		base64.StdEncoding.EncodeToString(backImage),
	}

	log.Printf("Calling Ollama API (model: %s) for IdentifyCoin...", o.modelName)
	return o.chat(ctx, prompt, images, stream)
}

// IdentifyFromSingleImage identifies a coin from a single image.
func (o *OllamaClient) IdentifyFromSingleImage(ctx context.Context, image []byte, stream chan<- string) (*models.CoinAnalysis, error) {
	if len(image) == 0 {
		if stream != nil {
			close(stream)
		}
		return nil, fmt.Errorf("image cannot be empty")
	}

	prompt := "Identify this coin from the image. Return ONLY a valid JSON object with the following fields: name, description, year, country, universal_id. The 'universal_id' must be a numeric identifier like Numista ID if available (return ONLY the ID value, e.g., '325206' instead of 'Numista 325206'), otherwise return an empty string. Do not include markdown code blocks or any text outside the JSON object."

	images := []string{
		base64.StdEncoding.EncodeToString(image),
	}

	log.Printf("Calling Ollama API (model: %s) for IdentifyFromSingleImage...", o.modelName)
	return o.chat(ctx, prompt, images, stream)
}

// chat sends a request to the Ollama /api/chat endpoint and parses the CoinAnalysis response.
func (o *OllamaClient) chat(ctx context.Context, prompt string, images []string, stream chan<- string) (*models.CoinAnalysis, error) {
	if stream != nil {
		defer close(stream)
	}

	reqBody := ollamaChatRequest{
		Model: o.modelName,
		Messages: []ollamaMessage{
			{
				Role:    "user",
				Content: prompt,
				Images:  images,
			},
		},
		Stream: true,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ollama request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/api/chat", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	var fullText string
	dec := json.NewDecoder(resp.Body)
	for {
		var ollamaResp ollamaChatResponse
		if err := dec.Decode(&ollamaResp); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to decode stream: %w", err)
		}

		chunk := ollamaResp.Message.Content
		fullText += chunk
		if stream != nil {
			stream <- chunk
		}
	}

	log.Printf("Ollama raw response: %s", fullText)

	// Cleanup markdown fences if the model wrapped the JSON anyway
	text := fullText
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	var result models.CoinAnalysis
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("failed to parse ollama response as CoinAnalysis: %w, text: %s", err, text)
	}

	log.Printf("Ollama identification successful: %s", result.Name)
	return &result, nil
}
