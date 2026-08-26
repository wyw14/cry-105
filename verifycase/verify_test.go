package verifycase

import (
	"context"
	"errors"
	"testing"
	"time"

	"orbitlink/internal/api"
	"orbitlink/internal/rf"
	"orbitlink/internal/weather"
)

type failingRF struct{}
func (failingRF) Inhibit(context.Context,string,string)(rf.InhibitProof,error){return rf.InhibitProof{},errors.New("inhibit relay offline")}
func (failingRF) VerifyInhibit(string,string)bool{return false}
type observingStower struct{called bool}
func (s *observingStower) Stow(string,string,bool)error{s.called=true;return nil}

func TestWindStowWaitsForRFInhibit(t *testing.T) {
	runtime, err := api.NewRuntime(t.TempDir())
	if err != nil { t.Fatal(err) }
	stower := &observingStower{}
	coordinator := weather.NewCoordinator(20, failingRF{}, stower, runtime.Passes, runtime.Audit)
	_, err = coordinator.Handle(context.Background(), weather.WindSample{StationID:"station",GustMetersPerSecond:28,ObservedAt:time.Now().UTC()}, "", "ANT-01", "RF-01")
	if err == nil { t.Fatal("failed inhibit unexpectedly succeeded") }
	if stower.called { t.Fatal("antenna motion began before RF inhibit was confirmed") }
}
