package scheduler

import (
	"fmt"
	"time"
)

type ReservationStatus string

const (
	Reserved ReservationStatus = "reserved"
	Active   ReservationStatus = "active"
	Released ReservationStatus = "released"
)

type Interval struct {
	ReservationID string            `json:"reservation_id"`
	PassID        string            `json:"pass_id"`
	AntennaID     string            `json:"antenna_id"`
	Start         time.Time         `json:"start"`
	End           time.Time         `json:"end"`
	Generation    int64             `json:"generation"`
	Status        ReservationStatus `json:"status"`
}

func (i Interval) Validate() error {
	if i.ReservationID == "" || i.PassID == "" || i.AntennaID == "" {
		return fmt.Errorf("reservation, pass and antenna identity are required")
	}
	if !i.End.After(i.Start) {
		return fmt.Errorf("reservation end must follow start")
	}
	if i.Generation < 1 {
		return fmt.Errorf("owner generation must be positive")
	}
	return nil
}

func (i Interval) Overlaps(other Interval) bool {
	return i.Start.Before(other.End) && other.Start.Before(i.End)
}

type ConflictError struct {
	AntennaID string
	Existing  Interval
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("antenna %s is reserved by pass %s", e.AntennaID, e.Existing.PassID)
}
