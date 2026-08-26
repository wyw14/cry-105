package satellite

import (
	"fmt"
	"strings"
	"time"
)

type Satellite struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	NORAD   int       `json:"norad"`
	Band    string    `json:"band"`
	Enabled bool      `json:"enabled"`
	Updated time.Time `json:"updated"`
}

func (s Satellite) Validate() error {
	if strings.TrimSpace(s.ID) == "" || strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("satellite identity is required")
	}
	if s.NORAD <= 0 {
		return fmt.Errorf("NORAD catalog number must be positive")
	}
	switch s.Band {
	case "S", "X", "Ka":
		return nil
	default:
		return fmt.Errorf("unsupported receive band %q", s.Band)
	}
}

type Ephemeris struct {
	SatelliteID string    `json:"satellite_id"`
	Revision    int64     `json:"revision"`
	Epoch       time.Time `json:"epoch"`
	Line1       string    `json:"line1"`
	Line2       string    `json:"line2"`
	ImportedAt  time.Time `json:"imported_at"`
}
