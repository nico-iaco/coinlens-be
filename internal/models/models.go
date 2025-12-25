package models

import (
	"time"

	"github.com/google/uuid"
)

type Coin struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Year        string    `json:"year"`
	Country     string    `json:"country"`
	CreatedAt   time.Time `json:"created_at"`
}

type CoinAnalysis struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Year        string `json:"year"`
	Country     string `json:"country"`
}
