package antenna

import (
	"fmt"
	"time"

	"orbitlink/internal/audit"
	"orbitlink/internal/scheduler"
)

type Executor struct {
	fleet *Fleet
	queue *scheduler.CommandQueue
	audit *audit.Trail
}

func NewExecutor(fleet *Fleet, queue *scheduler.CommandQueue, trail *audit.Trail) *Executor {
	return &Executor{fleet: fleet, queue: queue, audit: trail}
}

func (e *Executor) RunDue(now time.Time) ([]scheduler.PointingCommand, error) {
	commands := e.queue.Due(now)
	executed := make([]scheduler.PointingCommand, 0, len(commands))
	for _, command := range commands {
		if err := e.Execute(command); err != nil {
			return executed, err
		}
		executed = append(executed, command)
	}
	return executed, nil
}

func (e *Executor) Execute(command scheduler.PointingCommand) error {
	if !e.queue.IsExecutable(command) {
		return fmt.Errorf("pointing command %s belongs to a retired revision", command.ID)
	}
	e.fleet.mu.Lock()
	defer e.fleet.mu.Unlock()
	state, found := e.fleet.states[command.AntennaID]
	if !found {
		return fmt.Errorf("antenna %s not found", command.AntennaID)
	}
	if state.OwnerPassID != command.PassID {
		return fmt.Errorf("pointing command owner no longer controls antenna")
	}
	state.Azimuth = command.Azimuth
	state.Elevation = command.Elevation
	state.Mode = Tracking
	state.LastCommandAt = time.Now().UTC()
	if err := state.ValidatePosition(); err != nil {
		return err
	}
	if err := e.fleet.update(state); err != nil {
		return err
	}
	_, _ = e.audit.Record(audit.Event{Component: "antenna", Action: "pointing.executed", Subject: state.ID, Fields: map[string]any{"pass_id": command.PassID, "revision": command.Revision, "azimuth": command.Azimuth, "elevation": command.Elevation}})
	return nil
}
