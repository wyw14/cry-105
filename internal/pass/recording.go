package pass

import (
	"fmt"

	"orbitlink/internal/recorder"
)

type RecordingState struct {
	passes   *Service
	recorder *recorder.Manager
}

func NewRecordingState(passes *Service, manager *recorder.Manager) *RecordingState {
	return &RecordingState{passes: passes, recorder: manager}
}

func (s *RecordingState) TransferOwner(passID, segmentID, from, to string, expectedGeneration int64) (recorder.Segment, error) {
	value, found := s.passes.Get(passID)
	if !found || value.RecordingSegmentID != segmentID {
		return recorder.Segment{}, fmt.Errorf("recording is not active for pass %s", passID)
	}
	segment, err := s.recorder.Transfer(segmentID, from, to, expectedGeneration)
	if err != nil {
		return recorder.Segment{}, err
	}
	err = s.passes.update(passID, func(current *Pass) error {
		current.OwnerStation = segment.OwnerStation
		current.OwnerGeneration = segment.OwnerGeneration
		return nil
	})
	return segment, err
}

func (s *RecordingState) Apply(passID string, segment recorder.Segment) error {
	return s.passes.update(passID, func(current *Pass) error {
		if current.RecordingSegmentID != segment.ID {
			return fmt.Errorf("segment does not belong to pass")
		}
		current.OwnerStation = segment.OwnerStation
		current.OwnerGeneration = segment.OwnerGeneration
		return nil
	})
}
