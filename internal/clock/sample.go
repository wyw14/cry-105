package clock

import "time"

type Sample struct {
	ObservedAt time.Time     `json:"observed_at"`
	Offset     time.Duration `json:"offset"`
	Source     string        `json:"source"`
	Quality    string        `json:"quality"`
}

type Correction struct {
	ID        string        `json:"id"`
	AppliedAt time.Time     `json:"applied_at"`
	Delta     time.Duration `json:"delta"`
	Before    time.Duration `json:"before"`
	After     time.Duration `json:"after"`
	Source    string        `json:"source"`
}

type Observer func(Correction)
