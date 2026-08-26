package decoder

import (
	"fmt"
	"sync"

	"orbitlink/internal/audit"
	"orbitlink/internal/recorder"
)

type Session struct {
	mu          sync.Mutex
	passID      string
	segmentID   string
	attemptID   string
	epoch       int64
	assembler   *Assembler
	recorder    *recorder.Manager
	audit       *audit.Trail
	lostSymbols int
}

func NewSession(passID, segmentID string, frameSize int, manager *recorder.Manager, trail *audit.Trail) (*Session, error) {
	assembler, err := NewAssembler(frameSize)
	if err != nil {
		return nil, err
	}
	return &Session{passID: passID, segmentID: segmentID, assembler: assembler, recorder: manager, audit: trail}, nil
}

func (s *Session) BeginEpoch(attemptID string) (int64, error) {
	if attemptID == "" {
		return 0, fmt.Errorf("receiver attempt is required for decoder epoch")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if attemptID == s.attemptID {
		return s.epoch, nil
	}
	discarded := s.assembler.Reset()
	s.lostSymbols += discarded
	s.attemptID = attemptID
	s.epoch++
	_, _ = s.audit.Record(audit.Event{Component: "decoder", Action: "epoch.started", Subject: s.passID, Fields: map[string]any{"attempt_id": attemptID, "epoch": s.epoch, "discarded_symbols": discarded}})
	return s.epoch, nil
}

func (s *Session) Feed(attemptID string, symbols []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if attemptID != s.attemptID || s.epoch == 0 {
		return 0, fmt.Errorf("symbols belong to inactive receiver attempt")
	}
	frames := s.assembler.Push(s.epoch, symbols)
	for _, frame := range frames {
		if err := s.recorder.Buffer(s.segmentID, frame); err != nil {
			return 0, err
		}
	}
	return len(frames), nil
}

func (s *Session) Snapshot() (attemptID string, epoch int64, lostSymbols int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.attemptID, s.epoch, s.lostSymbols
}
