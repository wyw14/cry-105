package receiver

import (
	"context"
	"fmt"
	"sync"
	"time"

	"orbitlink/internal/audit"
)

type Unlocker struct {
	mu     sync.Mutex
	delays map[string]time.Duration
	audit  *audit.Trail
}

func NewUnlocker(trail *audit.Trail) *Unlocker {
	return &Unlocker{delays: make(map[string]time.Duration), audit: trail}
}

func (u *Unlocker) SetDelay(chainID string, delay time.Duration) {
	u.mu.Lock()
	u.delays[chainID] = delay
	u.mu.Unlock()
}

func (u *Unlocker) Unlock(ctx context.Context, chainID, passID string) error {
	u.mu.Lock()
	delay := u.delays[chainID]
	u.mu.Unlock()
	if chainID == "" || passID == "" {
		return fmt.Errorf("receiver chain and pass are required")
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		_, _ = u.audit.Record(audit.Event{Component: "receiver", Action: "unlock.timeout", Subject: chainID, Severity: audit.Warning, Fields: map[string]any{"pass_id": passID}})
		return fmt.Errorf("receiver unlock deadline exceeded: %w", ctx.Err())
	case <-timer.C:
		_, _ = u.audit.Record(audit.Event{Component: "receiver", Action: "unlocked", Subject: chainID, Fields: map[string]any{"pass_id": passID}})
		return nil
	}
}
