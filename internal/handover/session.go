package handover

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"orbitlink/internal/audit"
	"orbitlink/internal/pass"
	"orbitlink/internal/recorder"
)

type Session struct {
	ID                    string    `json:"id"`
	PassID                string    `json:"pass_id"`
	SegmentID             string    `json:"segment_id"`
	Source                string    `json:"source"`
	Destination           string    `json:"destination"`
	SourceGeneration      int64     `json:"source_generation"`
	DestinationGeneration int64     `json:"destination_generation"`
	TransferredAt         time.Time `json:"transferred_at,omitempty"`
	CleanedAt             time.Time `json:"cleaned_at,omitempty"`
}

type Coordinator struct {
	mu         sync.Mutex
	sessions   map[string]Session
	recordings *pass.RecordingState
	recorder   *recorder.Manager
	audit      *audit.Trail
}

func NewCoordinator(recordings *pass.RecordingState, manager *recorder.Manager, trail *audit.Trail) *Coordinator {
	return &Coordinator{sessions: make(map[string]Session), recordings: recordings, recorder: manager, audit: trail}
}

func (c *Coordinator) Begin(passID, segmentID, source, destination string, generation int64) (Session, error) {
	if passID == "" || segmentID == "" || source == "" || destination == "" || generation < 1 {
		return Session{}, fmt.Errorf("complete handover identity is required")
	}
	session := Session{ID: uuid.NewString(), PassID: passID, SegmentID: segmentID, Source: source, Destination: destination, SourceGeneration: generation}
	c.mu.Lock()
	c.sessions[session.ID] = session
	c.mu.Unlock()
	return session, nil
}

func (c *Coordinator) Transfer(sessionID string) (Session, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	session, found := c.sessions[sessionID]
	if !found {
		return Session{}, fmt.Errorf("handover session %s not found", sessionID)
	}
	segment, err := c.recordings.TransferOwner(session.PassID, session.SegmentID, session.Source, session.Destination, session.SourceGeneration)
	if err != nil {
		return Session{}, err
	}
	session.DestinationGeneration = segment.OwnerGeneration
	session.TransferredAt = time.Now().UTC()
	c.sessions[sessionID] = session
	_, _ = c.audit.Record(audit.Event{Component: "handover", Action: "transferred", Subject: session.ID, Fields: map[string]any{"pass_id": session.PassID, "destination_generation": session.DestinationGeneration}})
	return session, nil
}

func (c *Coordinator) Cleanup(sessionID string) error {
	c.mu.Lock()
	session, found := c.sessions[sessionID]
	if !found {
		c.mu.Unlock()
		return fmt.Errorf("handover session %s not found", sessionID)
	}
	session.CleanedAt = time.Now().UTC()
	c.sessions[sessionID] = session
	c.mu.Unlock()
	err := c.recorder.CloseForOwner(session.SegmentID, session.Source, session.SourceGeneration, "handover source cleanup")
	_, _ = c.audit.Record(audit.Event{Component: "handover", Action: "source.cleaned", Subject: session.ID, Fields: map[string]any{"close_error": errorText(err)}})
	return err
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
