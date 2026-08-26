package verifycase

import (
	"testing"
	"time"

	"orbitlink/internal/api"
	stationclock "orbitlink/internal/clock"
)

func TestClockCorrectionPreservesActivePassDuration(t *testing.T) {
	runtime, err := api.NewRuntime(t.TempDir())
	if err != nil { t.Fatal(err) }
	value := runtime.Passes.List()[0]
	before, err := runtime.Deadlines.Activate(value.ID, 2*time.Minute)
	if err != nil { t.Fatal(err) }
	if _, err := runtime.Clock.Apply(stationclock.Sample{
		ObservedAt: time.Now().UTC(), Offset: 7 * time.Second, Source: "gps", Quality: "locked",
	}); err != nil { t.Fatal(err) }
	after, found := runtime.Deadlines.Get(value.ID)
	if !found { t.Fatal("active deadline disappeared") }
	delta := after.MonotonicLOS.Sub(before.MonotonicLOS)
	if delta > 25*time.Millisecond || delta < -25*time.Millisecond {
		t.Fatalf("wall-clock correction shifted active monotonic deadline by %s", delta)
	}
}
