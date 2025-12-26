package handler

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"time"

	"coinlens-be/internal/database"
	"coinlens-be/internal/service"

	"github.com/google/uuid"
)

type CoinHandler struct {
	DB      *database.DB
	Gemini  *service.GeminiClient
	Storage *service.StorageService
}

func NewCoinHandler(db *database.DB, gemini *service.GeminiClient, storage *service.StorageService) *CoinHandler {
	return &CoinHandler{
		DB:      db,
		Gemini:  gemini,
		Storage: storage,
	}
}

func (h *CoinHandler) IdentifyCoin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Log the start of the request
	log.Printf("IdentifyCoin: Request received")

	// Parse multipart form
	// limit max memory to 10MB
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	frontFile, _, err := r.FormFile("front_image")
	if err != nil {
		http.Error(w, "Missing front_image", http.StatusBadRequest)
		return
	}
	defer frontFile.Close()

	backFile, _, err := r.FormFile("back_image")
	if err != nil {
		http.Error(w, "Missing back_image", http.StatusBadRequest)
		return
	}
	defer backFile.Close()

	// 1. Read files for Gemini (in memory)
	frontBytes, err := readFileToBytes(frontFile)
	if err != nil {
		http.Error(w, "Failed to read front image", http.StatusInternalServerError)
		return
	}
	log.Printf("Read front image: %d bytes", len(frontBytes))

	// seeking back to start for storage saving
	frontFile.Seek(0, 0)

	backBytes, err := readFileToBytes(backFile)
	if err != nil {
		http.Error(w, "Failed to read back image", http.StatusInternalServerError)
		return
	}
	backFile.Seek(0, 0)
	log.Printf("Read back image: %d bytes", len(backBytes))

	// 2. Call Gemini
	analysis, err := h.Gemini.IdentifyCoin(r.Context(), frontBytes, backBytes)
	if err != nil {
		log.Printf("Gemini error: %v", err)
		http.Error(w, "Failed to identify coin", http.StatusInternalServerError)
		return
	}
	log.Printf("Gemini analysis successful for coin: %s", analysis.Name)

	// Generate ID
	coinID := uuid.New()

	// 3. Save images to storage
	// Save files with deterministic names
	if err := h.Storage.SaveFile(frontFile, coinID.String()+"-front.jpg"); err != nil {
		log.Printf("Storage error front: %v", err)
		http.Error(w, "Failed to save images", http.StatusInternalServerError)
		return
	}

	if err := h.Storage.SaveFile(backFile, coinID.String()+"-back.jpg"); err != nil {
		log.Printf("Storage error back: %v", err)
		http.Error(w, "Failed to save images", http.StatusInternalServerError)
		return
	}

	// 4. Save to DB
	// We need a helper in database package or just exec here.
	// For simplicity, using raw SQL or pgx here.

	_, err = h.DB.Pool.Exec(context.Background(), `
        INSERT INTO coins (id, name, description, year, country)
        VALUES ($1, $2, $3, $4, $5)
    `, coinID, analysis.Name, analysis.Description, analysis.Year, analysis.Country)

	if err != nil {
		log.Printf("DB error: %v", err)
		http.Error(w, "Failed to save to database", http.StatusInternalServerError)
		return
	}

	log.Printf("Coin identified and saved successfully: %s", coinID.String())

	// 5. Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":          coinID.String(),
		"name":        analysis.Name,
		"description": analysis.Description,
		"year":        analysis.Year,
		"country":     analysis.Country,
	})
}

func (h *CoinHandler) GetCoins(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	log.Printf("GetCoins: Request received")

	rows, err := h.DB.Pool.Query(r.Context(), "SELECT id, name, description, year, country, created_at FROM coins ORDER BY created_at DESC")
	if err != nil {
		log.Printf("DB query error: %v", err)
		http.Error(w, "Failed to fetch coins", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var coins []map[string]interface{}
	for rows.Next() {
		var c struct {
			ID          string
			Name        string
			Description string
			Year        string
			Country     string
			CreatedAt   time.Time
		}
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.Year, &c.Country, &c.CreatedAt); err != nil {
			log.Printf("Row scan error: %v", err)
			continue
		}

		// Construct URLs deterministically
		frontURL := "/uploads/" + c.ID + "-front.jpg"
		backURL := "/uploads/" + c.ID + "-back.jpg"

		coins = append(coins, map[string]interface{}{
			"id":              c.ID,
			"name":            c.Name,
			"description":     c.Description,
			"year":            c.Year,
			"country":         c.Country,
			"image_front_url": frontURL,
			"image_back_url":  backURL,
			"created_at":      c.CreatedAt,
		})
	}
	log.Printf("GetCoins: Found %d coins", len(coins))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(coins)
}

func (h *CoinHandler) UpdateCoin(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, "Missing coin ID", http.StatusBadRequest)
		return
	}

	var payload struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if payload.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	var c struct {
		ID          string
		Name        string
		Description string
		Year        string
		Country     string
		CreatedAt   time.Time
	}

	err := h.DB.Pool.QueryRow(r.Context(),
		"UPDATE coins SET name = $1 WHERE id = $2 RETURNING id, name, description, year, country, created_at",
		payload.Name, idStr).Scan(&c.ID, &c.Name, &c.Description, &c.Year, &c.Country, &c.CreatedAt)

	if err != nil {
		log.Printf("DB update error: %v", err)
		// Check for no rows (coin not found) - pgx returns error for no rows in QueryRow
		if err.Error() == "no rows in result set" {
			http.Error(w, "Coin not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to update coin", http.StatusInternalServerError)
		return
	}

	// Construct URLs deterministically
	frontURL := "/uploads/" + c.ID + "-front.jpg"
	backURL := "/uploads/" + c.ID + "-back.jpg"

	response := map[string]interface{}{
		"id":              c.ID,
		"name":            c.Name,
		"description":     c.Description,
		"year":            c.Year,
		"country":         c.Country,
		"image_front_url": frontURL,
		"image_back_url":  backURL,
		"created_at":      c.CreatedAt,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func readFileToBytes(file multipart.File) ([]byte, error) {
	var buf []byte
	// Read all
	buf, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return buf, nil
}
