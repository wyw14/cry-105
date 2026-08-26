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
	// 强风处置必须先静默发射链：确认 RF 已禁止发射后，才允许天线离开跟踪路径。
	// 若在功放仍 enabled 时即驱动收拢，波束会在移动中扫出许可区域。
	proof, err := c.rf.Inhibit(ctx, chainID, result.OperationID)
	if err != nil {
		result.Error = err.Error()
		_, _ = c.audit.Record(audit.Event{Component: "weather", Action: "stow.inhibit-failed", Subject: antennaID, Severity: audit.Critical, Operation: result.OperationID, Fields: map[string]any{"error": err.Error()}})
		return result, fmt.Errorf("cannot confirm RF inhibit: %w", err)
	}
	result.InhibitConfirmed = proof.Confirmed && c.rf.VerifyInhibit(chainID, result.OperationID)
	if !result.InhibitConfirmed {
		result.Error = "RF inhibit proof does not match operation"
		_, _ = c.audit.Record(audit.Event{Component: "weather", Action: "stow.inhibit-failed", Subject: antennaID, Severity: audit.Critical, Operation: result.OperationID, Fields: map[string]any{"error": result.Error}})
		return result, fmt.Errorf("%s", result.Error)
	}
	// RF 已确认禁止发射，此时安全允许天线离开跟踪路径进入收拢。
	if err := c.antenna.Stow(antennaID, result.OperationID, result.InhibitConfirmed); err != nil {
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
