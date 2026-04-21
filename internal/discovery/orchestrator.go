// Package discovery — MODULE 1: Discovery Orchestrator (PRD).
//
// Independent sources run in parallel; pool-dependent sources run in parallel after merge.
// One source failure does not stop others.
package discovery

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"salesradar/internal/apollo"
	"salesradar/internal/domain"
	"salesradar/internal/googlesearch"
)

// ErrGoogleNotConfigured is returned when Google CSE credentials are missing.
var ErrGoogleNotConfigured = errors.New("google custom search API not configured")

// ErrApolloNotConfigured is returned when Apollo credentials are missing.
var ErrApolloNotConfigured = errors.New("apollo API not configured")

// DiscoverySource is the PRD contract for a registered discovery module.
type DiscoverySource interface {
	Name() string
	Run(ctx context.Context, p domain.RunParams) ([]domain.RawCandidate, error)
}

// --- Phase-1 sources (no prior candidate pool) ---

type seedDiscoverySource struct{}

func (seedDiscoverySource) Name() string { return sourceSeed }

func (seedDiscoverySource) Run(ctx context.Context, p domain.RunParams) ([]domain.RawCandidate, error) {
	rows, err := discoverSeed(ctx, p)
	if err != nil {
		return nil, err
	}
	// Module 2 guardrail: target at least 30 seed candidates when run capacity allows.
	target := seedMinimumTarget(BatchLimit(p))
	if len(rows) < target {
		refillParams := p
		refillParams.MaxLeadsThisRun = target
		refill, err := discoverSeed(ctx, refillParams)
		if err != nil {
			return nil, err
		}
		if len(refill) > target {
			refill = refill[:target]
		}
		rows = refill
	}
	return applyDirectoryDiscovery(rows, p), nil
}

type googleDiscoverySource struct {
	cfg     googlesearch.Config
	toggles domain.DiscoverySourceToggles
}

func (g googleDiscoverySource) Name() string { return sourceGoogle }

func (g googleDiscoverySource) Run(ctx context.Context, p domain.RunParams) ([]domain.RawCandidate, error) {
	if !g.cfg.Configured() {
		return nil, ErrGoogleNotConfigured
	}
	return discoverLive(ctx, g.cfg, p, g.toggles)
}

type apolloDiscoverySource struct{}

func (apolloDiscoverySource) Name() string { return sourceApollo }

func (apolloDiscoverySource) Run(ctx context.Context, p domain.RunParams) ([]domain.RawCandidate, error) {
	if strings.TrimSpace(apollo.APIKeyFromEnv()) == "" {
		return nil, ErrApolloNotConfigured
	}
	return discoverApollo(ctx, p)
}

// --- Pool-dependent sources (phase 2) ---

type jobPoolSource struct {
	pool  []domain.RawCandidate
	limit int
}

func (j jobPoolSource) Name() string { return sourceJob }

func (j jobPoolSource) Run(ctx context.Context, _ domain.RunParams) ([]domain.RawCandidate, error) {
	return runJobSignalSource(ctx, j.pool, j.limit), nil
}

// runSourcesParallel executes each DiscoverySource in its own goroutine and merges results.
// Failures are recorded per source; successful outputs are still merged.
func runSourcesParallel(ctx context.Context, sources []DiscoverySource, p domain.RunParams) (merged []domain.RawCandidate, statuses []ProviderStatus) {
	if len(sources) == 0 {
		return nil, nil
	}
	type result struct {
		name string
		out  []domain.RawCandidate
		err  error
	}
	results := make([]result, len(sources))
	var wg sync.WaitGroup
	for i, src := range sources {
		i, src := i, src
		wg.Add(1)
		go func() {
			defer wg.Done()
			out, err := src.Run(ctx, p)
			results[i] = result{name: src.Name(), out: out, err: err}
		}()
	}
	wg.Wait()

	for _, r := range results {
		st := ProviderStatus{ProviderName: r.name}
		if r.name == sourceApollo {
			st.ProviderLabel = "Apollo"
		}
		switch {
		case r.err != nil:
			if errors.Is(r.err, ErrGoogleNotConfigured) {
				st.State = ProviderNotConfigured
				st.SkipReason = "missing API key"
				st.ReasonCode = "provider_not_configured"
				st.ReasonMessage = "Provider is enabled but API key is missing."
			} else if errors.Is(r.err, ErrApolloNotConfigured) {
				st.State = ProviderNotConfigured
				st.Configured = boolPtr(false)
				st.SkipReason = "missing API key"
				st.ReasonCode = "provider_not_configured"
				st.ReasonMessage = "Provider is enabled but API key is missing."
			} else {
				st.State = ProviderFailed
				st.ReasonCode = "unknown_provider_error"
				st.ReasonMessage = "Provider execution failed."
			}
			st.LastError = r.err.Error()
			statuses = append(statuses, st)
		case len(r.out) == 0:
			st.State = ProviderSkipped
			st.ReasonCode = "no_output"
			st.ReasonMessage = "Provider executed but produced no output."
			st.LastError = emptyOutputReason(r.name)
			st.CandidatesTotal = 0
			statuses = append(statuses, st)
		default:
			st.State = ProviderSuccess
			if r.name == sourceApollo {
				st.Configured = boolPtr(true)
			}
			st.CandidatesTotal = len(r.out)
			st.CandidatesSuccess = len(r.out)
			st.Details = map[string]any{
				"raw_candidates": len(r.out),
			}
			statuses = append(statuses, st)
			for i := range r.out {
				c := &r.out[i]
				if !containsTrace(c.ProspectTrace.SourceTrace, r.name) {
					c.ProspectTrace.SourceTrace = append([]string{r.name}, c.ProspectTrace.SourceTrace...)
				}
				merged = append(merged, *c)
			}
		}
	}
	return merged, statuses
}

// runPoolSourcesParallel runs the job-signal phase-2 source when eligible; website crawl is handled
// in applyWebsiteEnrichmentToPool (in-place) so it still runs when the batch is already full.
func runPoolSourcesParallel(ctx context.Context, p domain.RunParams, pool []domain.RawCandidate, limit int, jobEnv bool, mode string) (extra []domain.RawCandidate, statuses []ProviderStatus) {
	tog := domain.SourceTogglesOrDefault(p.SourceToggles)
	jobEnabled := jobEnv && tog.JobSignal

	var sources []DiscoverySource
	jobEligible := countJobEligible(pool)
	if isMultiLike(mode) && jobEnabled && jobEligible > 0 {
		sources = append(sources, jobPoolSource{pool: pool, limit: limit})
	} else {
		st := ProviderStatus{ProviderName: sourceJob, State: ProviderSkipped}
		switch {
		case !isMultiLike(mode):
			st.SkipReason = "disabled by config"
			st.ReasonCode = "disabled_by_config"
			st.ReasonMessage = "Provider disabled by runtime discovery mode."
		case !jobEnv:
			st.SkipReason = "disabled by config"
			st.ReasonCode = "disabled_by_config"
			st.ReasonMessage = "Provider disabled by environment flag."
		case !tog.JobSignal:
			st.SkipReason = "disabled by settings"
			st.State = ProviderDisabled
			st.ReasonCode = "disabled_by_settings"
			st.ReasonMessage = "Provider disabled in discovery settings."
		default:
			st.SkipReason = fmt.Sprintf("no eligible candidates (eligible=%d/%d)", jobEligible, len(pool))
			st.ReasonCode = "no_eligible_candidates"
			st.ReasonMessage = "No candidates are eligible for this provider."
		}
		statuses = append(statuses, st)
	}
	if len(sources) == 0 {
		return nil, statuses
	}
	out, st2 := runSourcesParallel(ctx, sources, p)
	return out, append(statuses, st2...)
}

func seedMinimumTarget(batchLimit int) int {
	if batchLimit <= 0 {
		return 0
	}
	if batchLimit < 30 {
		return batchLimit
	}
	return 30
}

func emptyOutputReason(source string) string {
	switch source {
	case sourceWebsite:
		return "no website-qualified candidates emitted"
	case sourceJob:
		return "no job-signal candidates emitted"
	case sourceApollo:
		return "no Apollo organizations returned"
	default:
		return ""
	}
}

// buildPhase1SourcesWithSkips returns phase-1 sources and explicit skip rows for sources disabled in settings.
func buildPhase1SourcesWithSkips(mode string, googleCfg googlesearch.Config, t domain.DiscoverySourceToggles) (sources []DiscoverySource, skipped []ProviderStatus) {
	disabled := func(name string) bool {
		skipped = append(skipped, ProviderStatus{
			ProviderName:  name,
			State:         ProviderDisabled,
			SkipReason:    "disabled by settings",
			ReasonCode:    "disabled_by_settings",
			ReasonMessage: "Provider disabled in discovery settings.",
		})
		return true
	}
	switch mode {
	case "seed_only":
		if t.Seed {
			sources = append(sources, seedDiscoverySource{})
		} else {
			disabled(sourceSeed)
			// Without seed, "seed_only" must still run a primary web source; otherwise phase-1 is empty.
			if t.Google {
				sources = append(sources, googleDiscoverySource{cfg: googleCfg, toggles: t})
			} else {
				disabled(sourceGoogle)
			}
		}
	case "google_first":
		if t.Seed {
			sources = append(sources, seedDiscoverySource{})
		} else {
			disabled(sourceSeed)
		}
		if t.Google {
			sources = append(sources, googleDiscoverySource{cfg: googleCfg, toggles: t})
		} else {
			disabled(sourceGoogle)
		}
		if t.Apollo {
			sources = append(sources, apolloDiscoverySource{})
		} else {
			disabled(sourceApollo)
		}
	default:
		if t.Seed {
			sources = append(sources, seedDiscoverySource{})
		} else {
			disabled(sourceSeed)
		}
		if t.Google {
			sources = append(sources, googleDiscoverySource{cfg: googleCfg, toggles: t})
		} else {
			disabled(sourceGoogle)
		}
		if t.Apollo {
			sources = append(sources, apolloDiscoverySource{})
		} else {
			disabled(sourceApollo)
		}
	}
	return sources, skipped
}

func isMultiLike(mode string) bool {
	return mode == "multi_source" || mode == "google_first"
}

func normalizeMode(m string) string {
	m = strings.ToLower(strings.TrimSpace(m))
	if m == "" {
		return "multi_source"
	}
	switch m {
	case "seed_only", "google_first", "multi_source":
		return m
	default:
		return "multi_source"
	}
}

// augmentProviderStatuses ensures debug always lists every PRD source (no silent omission).
func augmentProviderStatuses(mode string, in []ProviderStatus) []ProviderStatus {
	out := append([]ProviderStatus(nil), in...)
	seen := map[string]bool{}
	for _, p := range out {
		seen[p.ProviderName] = true
	}
	if mode == "seed_only" {
		add := func(name, reason string) {
			if seen[name] {
				return
			}
			out = append(out, ProviderStatus{
				ProviderName:  name,
				State:         ProviderDisabled,
				SkipReason:    reason,
				ReasonCode:    "disabled_by_config",
				ReasonMessage: "Provider disabled by runtime discovery mode.",
			})
			seen[name] = true
		}
		add(sourceGoogle, "disabled by config")
		add(sourceWebsite, "disabled by config")
		add(sourceJob, "disabled by config")
	}
	for _, name := range []string{sourceSeed, sourceGoogle, sourceWebsite, sourceJob} {
		if !seen[name] {
			out = append(out, ProviderStatus{
				ProviderName:  name,
				State:         ProviderSkipped,
				SkipReason:    "provider not implemented",
				ReasonCode:    "provider_not_implemented",
				ReasonMessage: "Provider is present in debug output but not implemented for this run.",
			})
			seen[name] = true
		}
	}
	return out
}
