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
	"coinlens-be/internal/models"
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
        INSERT INTO coins (id, name, description, year, country, universal_id)
        VALUES ($1, $2, $3, $4, $5, $6)
    `, coinID, analysis.Name, analysis.Description, analysis.Year, analysis.Country, analysis.UniversalID)

	if err != nil {
		log.Printf("DB error: %v", err)
		http.Error(w, "Failed to save to database", http.StatusInternalServerError)
		return
	}

	log.Printf("Coin identified and saved successfully: %s", coinID.String())

	// 5. Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":           coinID.String(),
		"name":         analysis.Name,
		"description":  analysis.Description,
		"year":         analysis.Year,
		"country":      analysis.Country,
		"universal_id": analysis.UniversalID,
	})
}

func (h *CoinHandler) GetCoins(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	log.Printf("GetCoins: Request received")

	rows, err := h.DB.Pool.Query(r.Context(), "SELECT id, name, description, year, country, COALESCE(universal_id, ''), created_at FROM coins ORDER BY created_at DESC")
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
			UniversalID string
			CreatedAt   time.Time
		}
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.Year, &c.Country, &c.UniversalID, &c.CreatedAt); err != nil {
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
			"universal_id":    c.UniversalID,
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
		UniversalID string
		CreatedAt   time.Time
	}

	err := h.DB.Pool.QueryRow(r.Context(),
		"UPDATE coins SET name = $1 WHERE id = $2 RETURNING id, name, description, year, country, COALESCE(universal_id, ''), created_at",
		payload.Name, idStr).Scan(&c.ID, &c.Name, &c.Description, &c.Year, &c.Country, &c.UniversalID, &c.CreatedAt)

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

func (h *CoinHandler) DeleteCoin(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, "Missing coin ID", http.StatusBadRequest)
		return
	}

	// 1. Delete from DB
	if err := h.DB.DeleteCoin(r.Context(), idStr); err != nil {
		log.Printf("DB delete error: %v", err)
		if err.Error() == "coin not found" || err.Error() == "failed to delete coin: coin not found" {
			http.Error(w, "Coin not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to delete coin", http.StatusInternalServerError)
		return
	}

	// 2. Delete images (best effort)
	// Construct filenames based on ID
	frontFilename := idStr + "-front.jpg"
	backFilename := idStr + "-back.jpg"

	if err := h.Storage.DeleteFile(frontFilename); err != nil {
		log.Printf("Failed to delete front image: %v", err)
		// We don't fail the request if image deletion fails, ensuring DB consistency is more important
	}
	if err := h.Storage.DeleteFile(backFilename); err != nil {
		log.Printf("Failed to delete back image: %v", err)
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CoinHandler) CreateCoin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// limit max memory to 10MB
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	description := r.FormValue("description")
	year := r.FormValue("year")
	country := r.FormValue("country")
	universalID := r.FormValue("universal_id")

	if name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	// Handle Images
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

	coinID := uuid.New()

	// Save images
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

	coin := &models.Coin{
		ID:          coinID,
		Name:        name,
		Description: description,
		Year:        year,
		Country:     country,
		UniversalID: universalID,
	}

	if err := h.DB.CreateCoin(r.Context(), coin); err != nil {
		log.Printf("DB error: %v", err)
		http.Error(w, "Failed to save coin to database", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(coin)
}

func (h *CoinHandler) ReidentifyCoin(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if idStr == "" {
		http.Error(w, "Missing coin ID", http.StatusBadRequest)
		return
	}

	// 1. Read files from storage
	frontBytes, err := h.Storage.ReadFile(idStr + "-front.jpg")
	if err != nil {
		http.Error(w, "Failed to read front image", http.StatusInternalServerError)
		return
	}

	backBytes, err := h.Storage.ReadFile(idStr + "-back.jpg")
	if err != nil {
		http.Error(w, "Failed to read back image", http.StatusInternalServerError)
		return
	}

	// 2. Call Gemini
	analysis, err := h.Gemini.IdentifyCoin(r.Context(), frontBytes, backBytes)
	if err != nil {
		log.Printf("Gemini error: %v", err)
		http.Error(w, "Failed to identify coin", http.StatusInternalServerError)
		return
	}

	// 3. Update DB
	if err := h.DB.UpdateCoinMetadata(r.Context(), idStr, analysis); err != nil {
		log.Printf("DB update error: %v", err)
		http.Error(w, "Failed to update coin metadata", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(analysis)
}
func (h *CoinHandler) SearchSimilarCoins(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("reverse_image")
	if err != nil {
		http.Error(w, "Missing reverse_image", http.StatusBadRequest)
		return
	}
	defer file.Close()

	imageBytes, err := readFileToBytes(file)
	if err != nil {
		http.Error(w, "Failed to read image", http.StatusInternalServerError)
		return
	}

	results, err := h.Gemini.IdentifyFromSingleImage(r.Context(), imageBytes)
	if err != nil {
		log.Printf("Gemini search error: %v", err)
		http.Error(w, "Failed to identify coin", http.StatusInternalServerError)
		return
	}

	// Optimization: if the coin has a universal_id, search the DB
	var dbMatches []models.Coin
	if results.UniversalID != "" {
		dbMatches, err = h.DB.GetCoinsByUniversalID(r.Context(), results.UniversalID)
		if err != nil {
			log.Printf("DB search error: %v", err)
			// We don't fail the request if DB search fails, just log it
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ai_analysis": results,
		"db_matches":  dbMatches,
	})
}
