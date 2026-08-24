package antenna

import (
	"fmt"
	"time"

	"orbitlink/internal/audit"
)

type StowController struct {
	fleet *Fleet
	audit *audit.Trail
}

func NewStowController(fleet *Fleet, trail *audit.Trail) *StowController {
	return &StowController{fleet: fleet, audit: trail}
}

func (c *StowController) Stow(antennaID, operationID string, inhibitConfirmed bool) error {
	if !inhibitConfirmed {
		return fmt.Errorf("RF inhibit confirmation is required before stow")
	}
	c.fleet.mu.Lock()
	defer c.fleet.mu.Unlock()
	state, found := c.fleet.states[antennaID]
	if !found {
		return fmt.Errorf("antenna %s not found", antennaID)
	}
	state.Mode = Stowing
	state.Azimuth = 180
	state.Elevation = 90
	state.LastCommandAt = time.Now().UTC()
	state.Mode = Stowed
	if err := c.fleet.update(state); err != nil {
		return err
	}
	_, _ = c.audit.Record(audit.Event{Component: "antenna", Action: "stowed", Subject: antennaID, Operation: operationID})
	return nil
}
