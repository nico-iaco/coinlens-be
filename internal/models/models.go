package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// FlexibleString unmarshals JSON values that may be either a string or a number.
// Some LLMs (e.g. Ollama models) return numeric fields like "year" as JSON numbers
// rather than strings.
type FlexibleString string

func (f *FlexibleString) UnmarshalJSON(data []byte) error {
	// Try string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*f = FlexibleString(s)
		return nil
	}
	// Fall back to number → convert to string
	var n json.Number
	if err := json.Unmarshal(data, &n); err == nil {
		*f = FlexibleString(n.String())
		return nil
	}
	return fmt.Errorf("FlexibleString: cannot unmarshal %s", string(data))
}

func (f FlexibleString) String() string { return string(f) }

type Coin struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Year        string    `json:"year"`
	Country     string    `json:"country"`
	UniversalID string    `json:"universal_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type CoinAnalysis struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Year        FlexibleString `json:"year"`
	Country     string         `json:"country"`
	UniversalID FlexibleString `json:"universal_id"`
}
