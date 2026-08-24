package pass

import (
	"fmt"
	"time"

	"orbitlink/internal/audit"
	"orbitlink/internal/store"
)

type WindowRecomputer struct {
	store *store.WindowStore
	audit *audit.Trail
}

func NewWindowRecomputer(windowStore *store.WindowStore, trail *audit.Trail) *WindowRecomputer {
	return &WindowRecomputer{store: windowStore, audit: trail}
}

func (r *WindowRecomputer) Apply(passID string, window Window) error {
	if err := window.Validate(); err != nil {
		return err
	}
	if passID == "" {
		return fmt.Errorf("pass ID is required")
	}
	record := store.WindowRecord{PassID: passID, AOS: window.AOS, LOS: window.LOS, Revision: window.Revision, Updated: time.Now().UTC()}
	if err := r.store.SaveBoundary(record); err != nil {
		return err
	}
	_, _ = r.audit.Record(audit.Event{Component: "pass", Action: "window.recomputed", Subject: passID, Fields: map[string]any{"revision": window.Revision, "aos": window.AOS, "los": window.LOS}})
	return nil
}
