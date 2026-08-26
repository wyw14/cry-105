package receiver

import (
	"fmt"
	"sort"
)

type DiversityPolicy struct {
	RequiredChains []string
}

type DiversityDecision struct {
	Ready    bool
	Epoch    int64
	Failures map[string]string
	Chains   []string
}

func (p DiversityPolicy) Evaluate(results []AcquisitionResult) (DiversityDecision, error) {
	if len(p.RequiredChains) == 0 {
		return DiversityDecision{}, fmt.Errorf("diversity policy requires at least one receiver")
	}
	byChain := make(map[string]AcquisitionResult, len(results))
	for _, result := range results {
		byChain[result.ChainID] = result
	}
	decision := DiversityDecision{Failures: make(map[string]string)}
	var epoch int64
	epochKnown := false
	for _, chainID := range p.RequiredChains {
		decision.Chains = append(decision.Chains, chainID)
		result, found := byChain[chainID]
		if !found {
			decision.Failures[chainID] = "pending"
			continue
		}
		if !result.Locked {
			decision.Failures[chainID] = failureCode(result.ErrorCode, "no-lock")
			continue
		}
		if !result.Synced {
			decision.Failures[chainID] = "SYNC-OFFSET"
			continue
		}
		if epochKnown && result.Epoch != epoch {
			decision.Failures[chainID] = "EPOCH-MISMATCH"
			continue
		}
		epoch = result.Epoch
		epochKnown = true
	}
	sort.Strings(decision.Chains)
	decision.Epoch = epoch
	decision.Ready = len(decision.Failures) == 0 && len(decision.Chains) == len(p.RequiredChains)
	return decision, nil
}

func failureCode(code, fallback string) string {
	if code == "" {
		return fallback
	}
	return code
}
