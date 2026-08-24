package scheduler_test

import (
	"sync"
	"testing"
	"time"

	"orbitlink/internal/audit"
	"orbitlink/internal/store"
	"orbitlink/internal/scheduler"
)

// TestExtendAndReserveCannotProduceOverlappingOwnership reproduces the
// concurrency defect where an in-progress pass extension and a successor
// reservation both adjudicated against a stale resource state and therefore
// both succeeded, leaving two passes owning the same antenna at the same
// time. The check-and-publish boundary must be held under one lock so that a
// successor Reserve observes the extended interval (and is rejected) before
// the active pass yields the antenna.
func TestExtendAndReserveCannotProduceOverlappingOwnership(t *testing.T) {
	for iteration := 0; iteration < 50; iteration++ {
		paths := store.NewPaths(t.TempDir())
		trail := audit.NewTrail(store.NewAppendLog(paths))
		timeline, err := scheduler.NewTimeline(paths, trail)
		if err != nil {
			t.Fatalf("iter %d: new timeline: %v", iteration, err)
		}

		// A12 occupies ANT-01 from AOS-30s until LOS.
		aos := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
		los := aos.Add(8 * time.Minute)
		antenna := "ANT-01"
		active, err := timeline.Reserve("SAT-A12", antenna, aos.Add(-30*time.Second), los)
		if err != nil {
			t.Fatalf("iter %d: reserve active: %v", iteration, err)
		}

		// The next satellite is scheduled to take ANT-01 the instant A12
		// leaves, and A12 is extended by three minutes for good signal at
		// exactly the same moment. Both control requests run concurrently.
		var (
			wg         sync.WaitGroup
			extendErr  error
			reserveErr error
		)
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, extendErr = timeline.Extend(active.ReservationID, los.Add(3*time.Minute))
		}()
		go func() {
			defer wg.Done()
			_, reserveErr = timeline.Reserve("SAT-NEXT", antenna, los, los.Add(8*time.Minute))
		}()
		wg.Wait()

		// At most one of the two conflicting mutations may succeed. If both
		// succeeded the antenna now has overlapping ownership, which is the
		// defect under repair.
		if extendErr == nil && reserveErr == nil {
			t.Fatalf("iter %d: both extend and reserve succeeded, antenna ownership overlaps", iteration)
		}

		// Whatever the interleaving, the published intervals must never
		// overlap for a non-released reservation on the same antenna.
		intervals := timeline.List(antenna)
		if overlap := findOverlap(intervals); overlap != nil {
			t.Fatalf("iter %d: overlapping ownership detected: %+v vs %+v", iteration, overlap[0], overlap[1])
		}

		// The owner at the overlap point must be unambiguous: exactly one
		// non-released reservation covers the contested instant (LOS + 90s),
		// not both the active pass and its successor.
		owners := ownersAt(intervals, los.Add(90*time.Second))
		if len(owners) != 1 {
			t.Fatalf("iter %d: expected exactly one owner at LOS+90s, got %d: %+v", iteration, len(owners), owners)
		}
	}
}

// TestExtendRejectsSuccessorThatAlreadyPublished guards the symmetric case:
// once the successor reservation is published, the active pass extension must
// observe it and refuse to overlap the successor window.
func TestExtendRejectsSuccessorThatAlreadyPublished(t *testing.T) {
	paths := store.NewPaths(t.TempDir())
	trail := audit.NewTrail(store.NewAppendLog(paths))
	timeline, err := scheduler.NewTimeline(paths, trail)
	if err != nil {
		t.Fatal(err)
	}
	aos := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	los := aos.Add(8 * time.Minute)
	antenna := "ANT-01"
	active, err := timeline.Reserve("SAT-A12", antenna, aos.Add(-30*time.Second), los)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := timeline.Reserve("SAT-NEXT", antenna, los, los.Add(8*time.Minute)); err != nil {
		t.Fatalf("successor reserve: %v", err)
	}
	if _, err := timeline.Extend(active.ReservationID, los.Add(3*time.Minute)); err == nil {
		t.Fatal("extend into published successor unexpectedly succeeded")
	}
	intervals := timeline.List(antenna)
	if overlap := findOverlap(intervals); overlap != nil {
		t.Fatalf("overlapping ownership detected: %+v vs %+v", overlap[0], overlap[1])
	}
}

func findOverlap(intervals []scheduler.Interval) []scheduler.Interval {
	for i := 0; i < len(intervals); i++ {
		if intervals[i].Status == scheduler.Released {
			continue
		}
		for j := i + 1; j < len(intervals); j++ {
			if intervals[j].Status == scheduler.Released {
				continue
			}
			if intervals[i].Overlaps(intervals[j]) {
				return []scheduler.Interval{intervals[i], intervals[j]}
			}
		}
	}
	return nil
}

// ownersAt returns every non-released interval covering the instant, which
// is the real invariant: exactly one pass may own the antenna at any time.
func ownersAt(intervals []scheduler.Interval, at time.Time) []scheduler.Interval {
	var owners []scheduler.Interval
	for _, value := range intervals {
		if value.Status == scheduler.Released {
			continue
		}
		if !at.Before(value.Start) && at.Before(value.End) {
			owners = append(owners, value)
		}
	}
	return owners
}
