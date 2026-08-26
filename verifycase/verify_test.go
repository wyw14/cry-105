package verifycase

import (
	"testing"

	"orbitlink/internal/api"
	"orbitlink/internal/receiver"
)

func TestDiversityTrackingRequiresSynchronizedReceivers(t *testing.T) {
	runtime, err := api.NewRuntime(t.TempDir())
	if err != nil { t.Fatal(err) }
	value := runtime.Passes.List()[0]
	attempt, err := runtime.Attempts.Begin(value.ID, "RF-01")
	if err != nil { t.Fatal(err) }
	if err := runtime.Passes.BeginAcquisition(value.ID, attempt.ID); err != nil { t.Fatal(err) }
	decision, err := runtime.Passes.AcceptDiversity(value.ID, []receiver.AcquisitionResult{
		{PassID: value.ID, AttemptID: attempt.ID, ChainID: "RF-01", Locked: true, Synced: true, Epoch: 9},
		{PassID: value.ID, AttemptID: attempt.ID, ChainID: "RF-02", Locked: true, Synced: false, Epoch: 8, ErrorCode: "SYNC-OFFSET"},
	})
	if err != nil { t.Fatal(err) }
	if decision.Ready { t.Fatal("incomplete diversity set entered tracking") }
	current, _ := runtime.Passes.Get(value.ID)
	if current.Phase != "acquiring" { t.Fatalf("pass advanced to %s", current.Phase) }
	if _, found := runtime.Recorder.ActiveForPass(value.ID); found {
		t.Fatal("recorder started before required receivers synchronized")
	}
}
