package antenna

import (
	"fmt"
	"time"
)

type Mode string

const (
	Idle     Mode = "idle"
	Slewing  Mode = "slewing"
	Tracking Mode = "tracking"
	Stowing  Mode = "stowing"
	Stowed   Mode = "stowed"
	Fault    Mode = "fault"
)

type State struct {
	ID              string    `json:"id"`
	Mode            Mode      `json:"mode"`
	Azimuth         float64   `json:"azimuth"`
	Elevation       float64   `json:"elevation"`
	OwnerPassID     string    `json:"owner_pass_id,omitempty"`
	OwnerGeneration int64     `json:"owner_generation"`
	LastCommandAt   time.Time `json:"last_command_at"`
	FaultReason     string    `json:"fault_reason,omitempty"`
}

func (s State) ValidatePosition() error {
	if s.Azimuth < 0 || s.Azimuth >= 360 {
		return fmt.Errorf("azimuth %.2f is outside station limits", s.Azimuth)
	}
	if s.Elevation < 0 || s.Elevation > 90 {
		return fmt.Errorf("elevation %.2f is outside station limits", s.Elevation)
	}
	return nil
}
