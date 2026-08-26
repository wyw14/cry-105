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
	for _, chainID := range p.RequiredChains {
		result, found := byChain[chainID]
		if !found {
			decision.Failures[chainID] = "pending"
			continue
		}
		decision.Chains = append(decision.Chains, chainID)
		if !result.Locked {
			decision.Failures[chainID] = result.ErrorCode
			continue
		}
		if result.Locked {
			decision.Epoch = result.Epoch
			decision.Ready = true
			return decision, nil
		}
		if !result.Synced {
			decision.Failures[chainID] = "SYNC-OFFSET"
			continue
		}
		if epoch == 0 {
			epoch = result.Epoch
		} else if epoch != result.Epoch {
			decision.Failures[chainID] = "EPOCH-MISMATCH"
		}
	}
	sort.Strings(decision.Chains)
	decision.Epoch = epoch
	decision.Ready = len(decision.Failures) == 0 && len(decision.Chains) == len(p.RequiredChains)
	return decision, nil
}
