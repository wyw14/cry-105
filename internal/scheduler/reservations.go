package scheduler

import (
	"fmt"
	"time"

	"orbitlink/internal/store"
)

type ReservationPlanner struct {
	windows  *store.WindowStore
	timeline *Timeline
	leadTime time.Duration
}

func NewReservationPlanner(windows *store.WindowStore, timeline *Timeline, leadTime time.Duration) *ReservationPlanner {
	return &ReservationPlanner{windows: windows, timeline: timeline, leadTime: leadTime}
}

func (p *ReservationPlanner) ReservePass(passID, antennaID string) (Interval, error) {
	window, found := p.windows.Get(passID)
	if !found {
		return Interval{}, fmt.Errorf("pass window %s not found", passID)
	}
	if !window.LOS.After(window.AOS) {
		return Interval{}, fmt.Errorf("empty pass window %s", passID)
	}
	return p.timeline.Reserve(passID, antennaID, window.AOS.Add(-p.leadTime), window.LOS)
}

func (p *ReservationPlanner) Refresh(passID string) (Interval, error) {
	window, found := p.windows.Get(passID)
	if !found {
		return Interval{}, fmt.Errorf("pass window %s not found", passID)
	}
	for _, interval := range p.timeline.List("") {
		if interval.PassID == passID && interval.Status != Released {
			if _, err := p.timeline.Release(interval.ReservationID); err != nil {
				return Interval{}, err
			}
			return p.timeline.Reserve(passID, interval.AntennaID, window.AOS.Add(-p.leadTime), window.LOS)
		}
	}
	return Interval{}, fmt.Errorf("reservation for pass %s not found", passID)
}
