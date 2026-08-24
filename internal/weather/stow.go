package weather

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"orbitlink/internal/antenna"
	"orbitlink/internal/audit"
	passsvc "orbitlink/internal/pass"
	"orbitlink/internal/rf"
)

type RFInhibiter interface {
	Inhibit(context.Context, string, string) (rf.InhibitProof, error)
	VerifyInhibit(string, string) bool
}

type AntennaStower interface {
	Stow(string, string, bool) error
}

type Coordinator struct {
	threshold float64
	rf        RFInhibiter
	antenna   AntennaStower
	passes    *passsvc.Service
	audit     *audit.Trail
}

func NewCoordinator(threshold float64, controller RFInhibiter, stower AntennaStower, passes *passsvc.Service, trail *audit.Trail) *Coordinator {
	return &Coordinator{threshold: threshold, rf: controller, antenna: stower, passes: passes, audit: trail}
}

func (c *Coordinator) Handle(ctx context.Context, sample WindSample, passID, antennaID, chainID string) (StowResult, error) {
	result := StowResult{OperationID: uuid.NewString(), AntennaID: antennaID, ChainID: chainID}
	if sample.GustMetersPerSecond < c.threshold {
		return result, nil
	}
	proof, err := c.rf.Inhibit(ctx, chainID, result.OperationID)
	if err != nil {
		result.Error = err.Error()
		_, _ = c.audit.Record(audit.Event{Component: "weather", Action: "stow.inhibit-failed", Subject: antennaID, Severity: audit.Critical, Operation: result.OperationID, Fields: map[string]any{"error": err.Error()}})
		return result, fmt.Errorf("cannot confirm RF inhibit: %w", err)
	}
	result.InhibitConfirmed = proof.Confirmed && c.rf.VerifyInhibit(chainID, result.OperationID)
	if !result.InhibitConfirmed {
		result.Error = "RF inhibit proof does not match operation"
		return result, fmt.Errorf("%s", result.Error)
	}
	if err := c.antenna.Stow(antennaID, result.OperationID, true); err != nil {
		result.Error = err.Error()
		return result, err
	}
	result.Stowed = true
	if passID != "" {
		if err := c.passes.Fail(passID, "high wind safety stow"); err != nil {
			return result, err
		}
		result.PassFailed = true
	}
	_, _ = c.audit.Record(audit.Event{Component: "weather", Action: "stow.completed", Subject: antennaID, Operation: result.OperationID, Fields: map[string]any{"gust_mps": sample.GustMetersPerSecond}})
	return result, nil
}

var _ AntennaStower = (*antenna.StowController)(nil)
