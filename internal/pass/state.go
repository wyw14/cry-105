package pass

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"orbitlink/internal/audit"
	"orbitlink/internal/receiver"
	"orbitlink/internal/recorder"
	"orbitlink/internal/store"
)

type Service struct {
	mu       sync.RWMutex
	path     string
	passes   map[string]Pass
	recorder *recorder.Manager
	audit    *audit.Trail
}

func NewService(paths store.Paths, manager *recorder.Manager, trail *audit.Trail) (*Service, error) {
	service := &Service{path: paths.Snapshot("passes"), passes: make(map[string]Pass), recorder: manager, audit: trail}
	var passes []Pass
	found, err := store.ReadJSON(service.path, &passes)
	if err != nil {
		return nil, err
	}
	if found {
		for _, value := range passes {
			service.passes[value.ID] = value
		}
	}
	return service, nil
}

func (s *Service) Create(request CreateRequest) (Pass, error) {
	value := Pass{
		ID: uuid.NewString(), SatelliteID: request.SatelliteID, AntennaID: request.AntennaID,
		RFChains:               append([]string(nil), request.RFChains...),
		RequiredReceiverChains: append([]string(nil), request.RequiredReceiverChains...),
		Window:                 Window{AOS: request.AOS, LOS: request.LOS, Revision: request.Revision},
		Phase:                  Planned, OwnerStation: request.OwnerStation, OwnerGeneration: 1,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := value.Validate(); err != nil {
		return Pass{}, err
	}
	s.mu.Lock()
	s.passes[value.ID] = value
	if err := s.flushLocked(); err != nil {
		delete(s.passes, value.ID)
		s.mu.Unlock()
		return Pass{}, err
	}
	s.mu.Unlock()
	_, _ = s.audit.Record(audit.Event{Component: "pass", Action: "created", Subject: value.ID, Fields: map[string]any{"satellite_id": value.SatelliteID, "revision": value.Window.Revision}})
	return value, nil
}

func (s *Service) Get(id string) (Pass, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, found := s.passes[id]
	return value, found
}

func (s *Service) List() []Pass {
	s.mu.RLock()
	defer s.mu.RUnlock()
	values := make([]Pass, 0, len(s.passes))
	for _, value := range s.passes {
		values = append(values, value)
	}
	return values
}

func (s *Service) SetReservation(passID, reservationID string, generation int64) error {
	return s.update(passID, func(value *Pass) error {
		value.ReservationID = reservationID
		value.ReservationGeneration = generation
		return nil
	})
}

func (s *Service) BeginAcquisition(passID, attemptID string) error {
	return s.update(passID, func(value *Pass) error {
		if value.Phase != Planned && value.Phase != Acquiring {
			return fmt.Errorf("pass cannot acquire from phase %s", value.Phase)
		}
		value.Phase = Acquiring
		value.CurrentAttemptID = attemptID
		value.FailureReason = ""
		return nil
	})
}

func (s *Service) ApplyAcquisitionResult(result receiver.AcquisitionResult) error {
	s.mu.Lock()
	value, found := s.passes[result.PassID]
	if !found {
		s.mu.Unlock()
		return fmt.Errorf("pass %s not found", result.PassID)
	}
	if value.CurrentAttemptID != result.AttemptID {
		s.mu.Unlock()
		_, _ = s.audit.Record(audit.Event{Component: "pass", Action: "stale-acquisition.ignored", Subject: result.PassID, Severity: audit.Warning, Fields: map[string]any{"attempt_id": result.AttemptID, "current_attempt": value.CurrentAttemptID}})
		return nil
	}
	// A single receiver result must never publish tracking or start the
	// recorder. Diversity reception requires every required receiver chain to
	// be locked and synced before tracking is published (see AcceptDiversity).
	// Individual results merely accumulate; the pass stays Acquiring until
	// diversity converges or is explicitly failed.
	if value.Phase == Acquiring {
		if result.Locked && result.Synced {
			value.DecoderEpoch = result.Epoch
		} else if value.FailureReason == "" {
			value.FailureReason = result.ErrorCode
		}
	}
	value.UpdatedAt = time.Now().UTC()
	s.passes[value.ID] = value
	err := s.flushLocked()
	s.mu.Unlock()
	return err
}

func (s *Service) AcceptDiversity(passID string, results []receiver.AcquisitionResult) (receiver.DiversityDecision, error) {
	s.mu.Lock()
	value, found := s.passes[passID]
	if !found {
		s.mu.Unlock()
		return receiver.DiversityDecision{}, fmt.Errorf("pass %s not found", passID)
	}
	decision, err := (receiver.DiversityPolicy{RequiredChains: value.RequiredReceiverChains}).Evaluate(results)
	if err != nil {
		s.mu.Unlock()
		return decision, err
	}
	if !decision.Ready {
		s.mu.Unlock()
		return decision, nil
	}
	// Tracking and recorder startup must converge together: until every
	// required receiver chain is locked and synced, neither may publish. The
	// recorder is started only after the pass reaches Tracking so that no
	// frames are ever recorded against an unsynchronized diversity set.
	if value.Phase == Tracking && value.RecordingSegmentID != "" {
		// Already converged for this pass; keep the existing segment.
		s.mu.Unlock()
		return decision, nil
	}
	if value.Phase != Acquiring {
		s.mu.Unlock()
		return decision, fmt.Errorf("pass cannot enter tracking from %s", value.Phase)
	}
	value.Phase = Tracking
	value.DecoderEpoch = decision.Epoch
	value.FailureReason = ""
	value.UpdatedAt = time.Now().UTC()
	s.passes[passID] = value
	if err := s.flushLocked(); err != nil {
		// Roll back the phase so a later attempt can retry convergence.
		value.Phase = Acquiring
		s.passes[passID] = value
		s.mu.Unlock()
		return decision, err
	}
	ownerStation := value.OwnerStation
	ownerGeneration := value.OwnerGeneration
	s.mu.Unlock()

	segment, err := s.recorder.Start(passID, ownerStation, ownerGeneration)
	if err != nil {
		// Recorder could not start; undo tracking publication so the pass does
		// not linger in Tracking without an active recorder.
		_ = s.update(passID, func(current *Pass) error {
			if current.Phase == Tracking && current.RecordingSegmentID == "" {
				current.Phase = Acquiring
				current.DecoderEpoch = 0
			}
			return nil
		})
		return decision, err
	}
	if err := s.update(passID, func(current *Pass) error {
		current.RecordingSegmentID = segment.ID
		return nil
	}); err != nil {
		return decision, err
	}
	_, _ = s.audit.Record(audit.Event{Component: "pass", Action: "diversity.converged", Subject: passID, Fields: map[string]any{"epoch": decision.Epoch, "chains": decision.Chains, "segment_id": segment.ID}})
	return decision, nil
}

func (s *Service) Fail(passID, reason string) error {
	return s.update(passID, func(value *Pass) error {
		value.Phase = Failed
		value.FailureReason = reason
		return nil
	})
}

func (s *Service) Cancel(passID, reason string) error {
	return s.update(passID, func(value *Pass) error {
		if value.Phase == Completed || value.Phase == Sealed {
			return fmt.Errorf("completed pass cannot be canceled")
		}
		value.Phase = Canceled
		value.FailureReason = reason
		return nil
	})
}

func (s *Service) update(passID string, apply func(*Pass) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, found := s.passes[passID]
	if !found {
		return fmt.Errorf("pass %s not found", passID)
	}
	if err := apply(&value); err != nil {
		return err
	}
	value.UpdatedAt = time.Now().UTC()
	s.passes[passID] = value
	return s.flushLocked()
}

func (s *Service) flushLocked() error {
	values := make([]Pass, 0, len(s.passes))
	for _, value := range s.passes {
		values = append(values, value)
	}
	return store.WriteJSON(s.path, values)
}
