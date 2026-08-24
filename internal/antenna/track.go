package antenna

import (
	"fmt"
	"time"

	"orbitlink/internal/scheduler"
)

type TrackPoint struct {
	At        time.Time
	Azimuth   float64
	Elevation float64
}

func BuildCommands(passID, antennaID string, revision int64, points []TrackPoint) ([]scheduler.PointingCommand, error) {
	if len(points) < 2 {
		return nil, fmt.Errorf("pointing plan requires at least two track points")
	}
	commands := make([]scheduler.PointingCommand, 0, len(points))
	for index, point := range points {
		state := State{Azimuth: point.Azimuth, Elevation: point.Elevation}
		if err := state.ValidatePosition(); err != nil {
			return nil, fmt.Errorf("track point %d: %w", index, err)
		}
		if index > 0 && !point.At.After(points[index-1].At) {
			return nil, fmt.Errorf("track points must be strictly ordered")
		}
		commands = append(commands, scheduler.PointingCommand{
			PassID: passID, AntennaID: antennaID, Revision: revision, ExecuteAt: point.At,
			Azimuth: point.Azimuth, Elevation: point.Elevation,
		})
	}
	return commands, nil
}
