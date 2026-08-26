package satellite

import (
	"fmt"
	"sync"
	"time"

	"orbitlink/internal/audit"
	"orbitlink/internal/store"
)

type RevisionListener func(previous, current Ephemeris)

type EphemerisService struct {
	mu        sync.RWMutex
	path      string
	current   map[string]Ephemeris
	listeners []RevisionListener
	audit     *audit.Trail
}

func NewEphemerisService(paths store.Paths, trail *audit.Trail) (*EphemerisService, error) {
	service := &EphemerisService{
		path: paths.Snapshot("ephemerides"), current: make(map[string]Ephemeris), audit: trail,
	}
	var items []Ephemeris
	found, err := store.ReadJSON(service.path, &items)
	if err != nil {
		return nil, err
	}
	if found {
		for _, item := range items {
			service.current[item.SatelliteID] = item
		}
	}
	return service, nil
}

func (s *EphemerisService) Subscribe(listener RevisionListener) {
	s.mu.Lock()
	s.listeners = append(s.listeners, listener)
	s.mu.Unlock()
}

func (s *EphemerisService) Import(value Ephemeris) (Ephemeris, error) {
	if value.SatelliteID == "" || value.Revision < 1 || value.Epoch.IsZero() {
		return Ephemeris{}, fmt.Errorf("ephemeris identity, revision and epoch are required")
	}
	s.mu.Lock()
	previous := s.current[value.SatelliteID]
	if previous.Revision >= value.Revision {
		s.mu.Unlock()
		return Ephemeris{}, fmt.Errorf("ephemeris revision %d is not newer than %d", value.Revision, previous.Revision)
	}
	value.ImportedAt = time.Now().UTC()
	s.current[value.SatelliteID] = value
	if err := s.flushLocked(); err != nil {
		s.current[value.SatelliteID] = previous
		s.mu.Unlock()
		return Ephemeris{}, err
	}
	listeners := append([]RevisionListener(nil), s.listeners...)
	s.mu.Unlock()
	_, _ = s.audit.Record(audit.Event{Component: "satellite", Action: "ephemeris.imported", Subject: value.SatelliteID, Fields: map[string]any{"revision": value.Revision, "previous_revision": previous.Revision}})
	for _, listener := range listeners {
		listener(previous, value)
	}
	return value, nil
}

func (s *EphemerisService) Current(satelliteID string) (Ephemeris, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, found := s.current[satelliteID]
	return value, found
}

func (s *EphemerisService) List() []Ephemeris {
	s.mu.RLock()
	defer s.mu.RUnlock()
	values := make([]Ephemeris, 0, len(s.current))
	for _, value := range s.current {
		values = append(values, value)
	}
	return values
}

func (s *EphemerisService) flushLocked() error {
	values := make([]Ephemeris, 0, len(s.current))
	for _, value := range s.current {
		values = append(values, value)
	}
	return store.WriteJSON(s.path, values)
}
