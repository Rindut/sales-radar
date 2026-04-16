package discovery

import (
	"context"
	"fmt"
	"strings"

	"salesradar/internal/domain"
)

// applyGrowthSignalDetection annotates candidates with growth-phase indicators.
func applyGrowthSignalDetection(_ context.Context, candidates []domain.RawCandidate) []domain.RawCandidate {
	out := make([]domain.RawCandidate, len(candidates))
	for i := range candidates {
		out[i] = addGrowthSignal(candidates[i])
	}
	return out
}

func addGrowthSignal(c domain.RawCandidate) domain.RawCandidate {
	ctx := strings.ToLower(strings.TrimSpace(c.UnstructuredContext))
	if !hasAny(ctx,
		"expanding", "expansion", "hiring spike", "mass hiring", "funding", "series a", "series b", "series c",
		"new branch", "opening branch", "new outlet", "opening outlet", "scale-up", "growth phase",
	) {
		return c
	}
	c.UnstructuredContext = strings.TrimSpace(c.UnstructuredContext + "\n" +
		"@growth_signal_detected: true\n" +
		"@growth_signal_reason: Growth phase indicates scaling needs")
	if !containsTrace(c.ProspectTrace.SourceTrace, "growth_signal") {
		c.ProspectTrace.SourceTrace = append(c.ProspectTrace.SourceTrace, "growth_signal")
	}
	if !strings.Contains(strings.ToLower(c.UnstructuredContext), "growth phase indicates scaling needs") {
		c.UnstructuredContext += fmt.Sprintf("\n%s", "Growth phase indicates scaling needs")
	}
	return c
}

