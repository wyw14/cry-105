package pass

import (
	"fmt"
	"time"

	"orbitlink/internal/antenna"
	"orbitlink/internal/audit"
	"orbitlink/internal/scheduler"
)

type ExtensionService struct {
	passes   *Service
	timeline *scheduler.Timeline
	fleet    *antenna.Fleet
	audit    *audit.Trail
}

func NewExtensionService(passes *Service, timeline *scheduler.Timeline, fleet *antenna.Fleet, trail *audit.Trail) *ExtensionService {
	return &ExtensionService{passes: passes, timeline: timeline, fleet: fleet, audit: trail}
}

func (s *ExtensionService) Extend(passID string, duration time.Duration) (Pass, error) {
	if duration <= 0 || duration > 15*time.Minute {
		return Pass{}, fmt.Errorf("extension must be between zero and fifteen minutes")
	}
	value, found := s.passes.Get(passID)
	if !found {
		return Pass{}, fmt.Errorf("pass %s not found", passID)
	}
	if value.Phase != Tracking && value.Phase != Acquiring {
		return Pass{}, fmt.Errorf("pass %s cannot extend from %s", passID, value.Phase)
	}
	interval, err := s.timeline.Extend(value.ReservationID, value.Window.LOS.Add(duration))
	if err != nil {
		return Pass{}, err
	}
	if owner, ok := s.fleet.State(value.AntennaID); !ok || owner.OwnerPassID != passID || owner.OwnerGeneration != interval.Generation {
		return Pass{}, fmt.Errorf("antenna ownership changed during extension")
	}
	if err := s.passes.update(passID, func(current *Pass) error {
		current.Window.LOS = current.Window.LOS.Add(duration)
		return nil
	}); err != nil {
		return Pass{}, err
	}
	updated, _ := s.passes.Get(passID)
	_, _ = s.audit.Record(audit.Event{Component: "pass", Action: "extended", Subject: passID, Fields: map[string]any{"duration_seconds": duration.Seconds(), "new_los": updated.Window.LOS}})
	return updated, nil
}
