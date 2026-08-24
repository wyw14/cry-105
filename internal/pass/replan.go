package pass

import (
	"fmt"
	"time"

	"orbitlink/internal/antenna"
	"orbitlink/internal/audit"
	"orbitlink/internal/satellite"
	"orbitlink/internal/scheduler"
)

type Replanner struct {
	passes  *Service
	windows *WindowRecomputer
	queue   *scheduler.CommandQueue
	audit   *audit.Trail
}

func NewReplanner(passes *Service, windows *WindowRecomputer, queue *scheduler.CommandQueue, trail *audit.Trail) *Replanner {
	return &Replanner{passes: passes, windows: windows, queue: queue, audit: trail}
}

func (r *Replanner) ApplyRevision(previous, current satellite.Ephemeris) {
	if current.Revision <= previous.Revision {
		return
	}
	for _, value := range r.passes.List() {
		if value.SatelliteID != current.SatelliteID || value.Phase != Planned {
			continue
		}
		shift := time.Duration(current.Revision-previous.Revision) * time.Second
		window := Window{AOS: value.Window.AOS.Add(shift), LOS: value.Window.LOS.Add(shift), Revision: current.Revision}
		if err := r.windows.Apply(value.ID, window); err != nil {
			_, _ = r.audit.Record(audit.Event{Component: "pass", Action: "replan.failed", Subject: value.ID, Severity: audit.Warning, Fields: map[string]any{"error": err.Error()}})
			continue
		}
		points := []antenna.TrackPoint{
			{At: window.AOS.Add(-20 * time.Second), Azimuth: 120, Elevation: 10},
			{At: window.AOS, Azimuth: 135, Elevation: 20},
			{At: window.LOS, Azimuth: 240, Elevation: 8},
		}
		commands, err := antenna.BuildCommands(value.ID, value.AntennaID, current.Revision, points)
		if err != nil {
			continue
		}
		if err := r.queue.Replace(value.ID, current.Revision, commands); err != nil {
			continue
		}
		_ = r.passes.update(value.ID, func(pass *Pass) error {
			pass.Window = window
			return nil
		})
	}
}

func (r *Replanner) QueueInitial(passID string) error {
	value, found := r.passes.Get(passID)
	if !found {
		return fmt.Errorf("pass %s not found", passID)
	}
	points := []antenna.TrackPoint{
		{At: value.Window.AOS.Add(-20 * time.Second), Azimuth: 100, Elevation: 10},
		{At: value.Window.AOS, Azimuth: 125, Elevation: 25},
		{At: value.Window.LOS, Azimuth: 250, Elevation: 7},
	}
	commands, err := antenna.BuildCommands(value.ID, value.AntennaID, value.Window.Revision, points)
	if err != nil {
		return err
	}
	return r.queue.Replace(value.ID, value.Window.Revision, commands)
}
