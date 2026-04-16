// Package discovery performs lead discovery from configured sources.
//
// When SALESRADAR_GOOGLE_API_KEY and SALESRADAR_GOOGLE_CX are set, Discover runs
// Google Custom Search first, then optional Apollo enrichment by domain and LinkedIn URLs from Apollo.
// Set SALESRADAR_USE_MOCK_DISCOVERY=1 to force deterministic mock candidates (tests / offline).
//
// Apollo: optional SALESRADAR_APOLLO_API_KEY for org lookup — not used as a primary domain source.
package discovery

import (
	"context"
	"errors"
	"os"
	"strings"

	"salesradar/internal/apollo"
	"salesradar/internal/domain"
	"salesradar/internal/googlesearch"
	"salesradar/internal/normalization"
)

// ErrEmptySourceAllowlist means RunParams.SourceAllowlist was nil or empty.
// That is a configuration error; callers must fail fast instead of treating it as zero leads.
var ErrEmptySourceAllowlist = errors.New("discovery: source allowlist is empty")

type ProviderState string

const (
	ProviderActive        ProviderState = "active"
	ProviderSkipped       ProviderState = "skipped"
	ProviderError         ProviderState = "error"
	ProviderNotConfigured ProviderState = "not_configured"
)

// ProviderStatus is surfaced to run stats/debug UI.
type ProviderStatus struct {
	ProviderName string        `json:"provider_name"`
	State        ProviderState `json:"state"` // configured | unavailable | fallback
	SkipReason   string        `json:"skip_reason,omitempty"`
	LastError    string        `json:"last_error,omitempty"`
}

// DiscoverResult contains candidates plus provider debug statuses.
type DiscoverResult struct {
	Candidates []domain.RawCandidate
	Providers  []ProviderStatus
	Mode       string
	Source     string
}

const (
	sourceSeed       = "seed_discovery"
	sourceGoogle     = "google_discovery"
	sourceWebsite    = "website_crawl_discovery"
	sourceJob        = "job_signal_discovery"
	sourceDirectory  = "directory_discovery"
	sourceApollo     = "apollo_enrichment"
	sourceLinkedIn   = "linkedin_signal"
)

// Provider abstracts discovery sources so they can be swapped without changing the pipeline.
type Provider interface {
	Name() string
	Configured() bool
	Discover(ctx context.Context, p domain.RunParams) ([]domain.RawCandidate, error)
}

type googleProvider struct{ cfg googlesearch.Config }

func (g googleProvider) Name() string     { return "google" }
func (g googleProvider) Configured() bool { return g.cfg.Configured() }
func (g googleProvider) Discover(ctx context.Context, p domain.RunParams) ([]domain.RawCandidate, error) {
	return discoverLive(ctx, g.cfg, p, domain.SourceTogglesOrDefault(p.SourceToggles))
}

type mockProvider struct{}

func (m mockProvider) Name() string     { return "mock" }
func (m mockProvider) Configured() bool { return true }
func (m mockProvider) Discover(ctx context.Context, p domain.RunParams) ([]domain.RawCandidate, error) {
	return discoverMock(ctx, p)
}

type seedProvider struct{}

func (s seedProvider) Name() string     { return "seed" }
func (s seedProvider) Configured() bool { return true }
func (s seedProvider) Discover(ctx context.Context, p domain.RunParams) ([]domain.RawCandidate, error) {
	return discoverSeed(ctx, p)
}

// Discover returns up to BatchLimit(p) raw company candidates.
//
// Rules:
//   - If SourceAllowlist is nil or empty, returns ErrEmptySourceAllowlist.
//   - MaxLeadsThisRun ≤ 0 uses domain.MaxLeadsPerRunDefault (50).
//   - Values above domain.MaxLeadsPerRunCap are clamped to 100.
//   - Mock data is ICP-flavored (training/onboarding relevance)—not a generic famous-company list.
func Discover(ctx context.Context, p domain.RunParams) ([]domain.RawCandidate, error) {
	res, err := DiscoverWithStatus(ctx, p)
	if err != nil {
		return nil, err
	}
	return res.Candidates, nil
}

// DiscoverWithStatus discovers candidates and returns provider status debug information.
func DiscoverWithStatus(ctx context.Context, p domain.RunParams) (DiscoverResult, error) {
	if err := ctx.Err(); err != nil {
		return DiscoverResult{}, err
	}
	if len(p.SourceAllowlist) == 0 {
		return DiscoverResult{}, ErrEmptySourceAllowlist
	}

	mode := normalizeMode(os.Getenv("DISCOVERY_MODE"))
	googleCfg := googlesearch.ConfigFromEnv()
	websiteEnv := strings.TrimSpace(os.Getenv("SALESRADAR_ENABLE_WEBSITE_CRAWL")) != "0"
	jobEnv := strings.TrimSpace(os.Getenv("SALESRADAR_ENABLE_JOB_SIGNAL")) != "0"
	toggles := domain.SourceTogglesOrDefault(p.SourceToggles)

	// MODULE 1 — Discovery Orchestrator: phase-1 independent sources in parallel.
	phase1Sources, phase1Skipped := buildPhase1SourcesWithSkips(mode, googleCfg, toggles)
	phase1Out, phase1Statuses := runSourcesParallel(ctx, phase1Sources, p)
	phase1Statuses = append(phase1Skipped, phase1Statuses...)
	candidatePool := phase1Out
	if len(candidatePool) == 0 {
		dir := applyDirectoryDiscovery(nil, p)
		if len(dir) > 0 {
			candidatePool = dir
			phase1Statuses = append(phase1Statuses, ProviderStatus{
				ProviderName: sourceDirectory,
				State:        ProviderActive,
			})
		}
	}
	limit := BatchLimit(p)
	if len(candidatePool) > limit {
		candidatePool = candidatePool[:limit]
	}

	// Phase 2 — pool-dependent sources (website crawl, job signal) in parallel.
	remaining := limit - len(candidatePool)
	if remaining < 0 {
		remaining = 0
	}
	phase2Out, phase2Statuses := runPoolSourcesParallel(ctx, p, candidatePool, remaining, websiteEnv, jobEnv, mode)
	candidatePool = append(candidatePool, phase2Out...)
	if len(candidatePool) > limit {
		candidatePool = candidatePool[:limit]
	}

	// MODULE 5 — Normalization layer for consistent downstream shape.
	candidatePool = normalization.NormalizeCandidates(candidatePool)
	// MODULE 6 — Deduplication and entity resolution (domain + similar-name merge).
	candidatePool = normalization.DeduplicateCandidates(candidatePool)
	candidatePool = applyGrowthSignalDetection(ctx, candidatePool)

	providers := augmentProviderStatuses(mode, append(phase1Statuses, phase2Statuses...))
	providers = mergeIntegrationProviderStatuses(providers, p)
	firstActive := firstActiveSourceName(providers, candidatePool)

	return DiscoverResult{
		Candidates: candidatePool,
		Providers:  providers,
		Mode:       mode,
		Source:     firstActive,
	}, nil
}

func mergeIntegrationProviderStatuses(providers []ProviderStatus, p domain.RunParams) []ProviderStatus {
	t := domain.SourceTogglesOrDefault(p.SourceToggles)
	seen := map[string]bool{}
	for _, x := range providers {
		seen[x.ProviderName] = true
	}
	add := func(st ProviderStatus) {
		if seen[st.ProviderName] {
			return
		}
		providers = append(providers, st)
		seen[st.ProviderName] = true
	}
	apolloKey := strings.TrimSpace(apollo.APIKeyFromEnv())

	if !t.Apollo {
		add(ProviderStatus{ProviderName: sourceApollo, State: ProviderSkipped, SkipReason: "disabled by settings"})
	} else if apolloKey == "" {
		add(ProviderStatus{ProviderName: sourceApollo, State: ProviderNotConfigured, SkipReason: "missing API key"})
	} else {
		add(ProviderStatus{ProviderName: sourceApollo, State: ProviderActive})
	}

	if !t.LinkedIn {
		add(ProviderStatus{ProviderName: sourceLinkedIn, State: ProviderSkipped, SkipReason: "disabled by settings"})
	} else if !t.Apollo {
		add(ProviderStatus{ProviderName: sourceLinkedIn, State: ProviderSkipped, SkipReason: "requires Apollo enrichment to be enabled"})
	} else if apolloKey == "" {
		add(ProviderStatus{ProviderName: sourceLinkedIn, State: ProviderNotConfigured, SkipReason: "missing API key"})
	} else {
		add(ProviderStatus{ProviderName: sourceLinkedIn, State: ProviderActive})
	}

	return providers
}

func firstActiveSourceName(providers []ProviderStatus, pool []domain.RawCandidate) string {
	for _, ps := range providers {
		if ps.State == ProviderActive {
			return ps.ProviderName
		}
	}
	if len(pool) > 0 {
		return pool[0].PrimaryDiscoverySourceName()
	}
	return sourceSeed
}

func discoverMock(ctx context.Context, p domain.RunParams) ([]domain.RawCandidate, error) {
	limit := BatchLimit(p)
	out := make([]domain.RawCandidate, 0, limit)
	for i := 0; i < limit; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		src := p.SourceAllowlist[i%len(p.SourceAllowlist)]
		out = append(out, mockCandidateAt(i, src))
	}
	return out, nil
}

func discoverSeed(ctx context.Context, p domain.RunParams) ([]domain.RawCandidate, error) {
	limit := BatchLimit(p)
	out := make([]domain.RawCandidate, 0, limit)
	for i := 0; i < limit; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		src := p.SourceAllowlist[i%len(p.SourceAllowlist)]
		out = append(out, seedCandidateAt(i, src))
	}
	return out, nil
}

// BatchLimit is the effective candidate ceiling for this run (default 50, hard cap 100).
func BatchLimit(p domain.RunParams) int {
	n := p.MaxLeadsThisRun
	if n <= 0 {
		n = domain.MaxLeadsPerRunDefault
	}
	if n > domain.MaxLeadsPerRunCap {
		n = domain.MaxLeadsPerRunCap
	}
	return n
}

func limitAndTag(in []domain.RawCandidate, limit int, sourceTag string) []domain.RawCandidate {
	if limit <= 0 || len(in) == 0 {
		return nil
	}
	out := make([]domain.RawCandidate, 0, len(in))
	for _, c := range in {
		if len(out) >= limit {
			break
		}
		if !containsTrace(c.ProspectTrace.SourceTrace, sourceTag) {
			c.ProspectTrace.SourceTrace = append([]string{sourceTag}, c.ProspectTrace.SourceTrace...)
		}
		out = append(out, c)
	}
	return out
}

func runWebsiteCrawlSource(ctx context.Context, pool []domain.RawCandidate, limit int) []domain.RawCandidate {
	if limit <= 0 || len(pool) == 0 {
		return nil
	}
	out := make([]domain.RawCandidate, 0, limit)
	for i := range pool {
		if len(out) >= limit {
			break
		}
		base := pool[i]
		enriched := enrichWithWebsiteCrawl(ctx, base)
		for _, c := range enriched {
			if len(out) >= limit {
				break
			}
			if !containsTrace(c.ProspectTrace.SourceTrace, sourceWebsite) {
				c.ProspectTrace.SourceTrace = append([]string{sourceWebsite}, c.ProspectTrace.SourceTrace...)
			}
			out = append(out, c)
		}
	}
	return out
}

func runJobSignalSource(ctx context.Context, pool []domain.RawCandidate, limit int) []domain.RawCandidate {
	if limit <= 0 || len(pool) == 0 {
		return nil
	}
	out := make([]domain.RawCandidate, 0, limit)
	for i := range pool {
		if len(out) >= limit {
			break
		}
		c := enrichWithJobSignal(ctx, pool[i])
		if !containsTrace(c.ProspectTrace.SourceTrace, "job_signal") {
			continue
		}
		if !containsTrace(c.ProspectTrace.SourceTrace, sourceJob) {
			c.ProspectTrace.SourceTrace = append([]string{sourceJob}, c.ProspectTrace.SourceTrace...)
		}
		out = append(out, c)
	}
	return out
}

func countDomainEligible(pool []domain.RawCandidate) int {
	n := 0
	for _, c := range pool {
		if strings.TrimSpace(c.OfficialDomain) != "" {
			n++
		}
	}
	return n
}

func countJobEligible(pool []domain.RawCandidate) int {
	n := 0
	for _, c := range pool {
		if strings.TrimSpace(c.OfficialDomain) != "" || strings.TrimSpace(companyNameFromContext(c.UnstructuredContext)) != "" {
			n++
		}
	}
	return n
}
