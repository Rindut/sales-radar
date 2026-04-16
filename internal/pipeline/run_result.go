package pipeline

import (
	"context"
	"strings"

	"salesradar/internal/deduplication"
	"salesradar/internal/discovery"
	"salesradar/internal/domain"
	"salesradar/internal/enrichment"
	"salesradar/internal/extraction"
	"salesradar/internal/icp"
	"salesradar/internal/odoo"
	"salesradar/internal/quality"
	"salesradar/internal/review"
	"salesradar/internal/status"
)

// RunStats summarizes a pipeline run for UI and CLI.
type RunStats struct {
	CandidatesFound   int `json:"candidates_found"`
	Enriched          int `json:"enriched"`
	ContactReady      int `json:"contact_ready"`
	ResearchFirst     int `json:"research_first"`
	Rejected          int `json:"rejected"`
	DuplicatesRemoved int `json:"duplicates_removed"`
	SemanticMerged    int `json:"semantic_merged"`
	RowsStored        int `json:"rows_stored"`
	// IntegrationUsed* is true if discovery/enrichment touched that integration for at least one candidate this run.
	IntegrationGoogleUsed   bool `json:"integration_google_used"`
	IntegrationApolloUsed   bool `json:"integration_apollo_used"`
	IntegrationLinkedInUsed bool `json:"integration_linkedin_used"`
	ProviderStatuses        []discovery.ProviderStatus `json:"provider_statuses"`
	DiscoveryMode           string                     `json:"discovery_mode"`
	DiscoverySource         string            `json:"discovery_source"`
	SourceBreakdown         []SourceBreakdown `json:"source_breakdown"`
	BreakdownGeneratedTotal int               `json:"breakdown_generated_total"`
	BreakdownMatchesTotal   bool              `json:"breakdown_matches_total"`
}

// SourceBreakdown reports discovery contribution by source.
type SourceBreakdown struct {
	SourceName string `json:"source_name"`
	Generated  int    `json:"generated"`
	Kept       int    `json:"kept"`
	Qualified  int    `json:"qualified"`
	Rejected   int    `json:"rejected"`
}

// PreparedRow is a staged lead plus review payload for persistence.
type PreparedRow struct {
	Staged domain.StagedOdooLead
	Review review.ReviewLead
	SourceName string
}

// RunWithQualityGate runs the pipeline, applies storage rules, and returns rows to persist plus stats.
func RunWithQualityGate(ctx context.Context, params domain.RunParams) ([]PreparedRow, RunStats, error) {
	disRes, err := discovery.DiscoverWithStatus(ctx, params)
	if err != nil {
		return nil, RunStats{}, err
	}
	raw := disRes.Candidates

	store := noopDedupStore{}
	client := noopOdooClient{}

	var stats RunStats
	stats.ProviderStatuses = append([]discovery.ProviderStatus(nil), disRes.Providers...)
	stats.DiscoveryMode = disRes.Mode
	stats.DiscoverySource = disRes.Source
	stats.CandidatesFound = len(raw)
	icpCfg := params.ICPRuntime
	if icpCfg == nil {
		icpCfg = domain.DefaultICPRuntimeSettings()
	}
	breakdown := map[string]*SourceBreakdown{}
	for _, c := range raw {
		src := sourceNameFromCandidate(c)
		sb := breakdown[src]
		if sb == nil {
			sb = &SourceBreakdown{SourceName: src}
			breakdown[src] = sb
		}
		sb.Generated++
		if c.ProspectTrace.UsedGoogle {
			stats.IntegrationGoogleUsed = true
		}
		if c.ProspectTrace.UsedApollo {
			stats.IntegrationApolloUsed = true
		}
		if c.ProspectTrace.UsedLinkedIn {
			stats.IntegrationLinkedInUsed = true
		}
	}

	var out []PreparedRow

	for _, c := range raw {
		ext, err := extraction.Extract(ctx, c)
		if err != nil {
			return nil, RunStats{}, err
		}
		enrichment.Apply(ctx, ext)
		icpLead, err := icp.Evaluate(ctx, ext, icpCfg)
		if err != nil {
			return nil, RunStats{}, err
		}
		deduped, err := deduplication.Classify(ctx, icpLead, store)
		if err != nil {
			return nil, RunStats{}, err
		}
		staged, err := status.AssignStatus(ctx, deduped)
		if err != nil {
			return nil, RunStats{}, err
		}
		_, err = odoo.Push(ctx, staged, client)
		if err != nil {
			return nil, RunStats{}, err
		}

		if staged.DuplicateStatus == domain.DupExact {
			stats.DuplicatesRemoved++
			breakdown[sourceNameFromCandidate(c)].Rejected++
			continue
		}
		if staged.ICPMatch == domain.ICPNo {
			stats.Rejected++
			breakdown[sourceNameFromCandidate(c)].Rejected++
			continue
		}

		rl := review.BuildReviewLead(*staged)
		if review.HasEnrichmentSignals(rl) {
			stats.Enriched++
		}

		if !quality.PassesStorageGate(rl) {
			stats.Rejected++
			breakdown[sourceNameFromCandidate(c)].Rejected++
			continue
		}

		rl = review.ApplySalesStatusAndCopy(*staged, rl)
		if rl.Action == review.ActionIgnore {
			stats.Rejected++
			breakdown[sourceNameFromCandidate(c)].Rejected++
			continue
		}
		breakdown[sourceNameFromCandidate(c)].Kept++
		out = append(out, PreparedRow{Staged: *staged, Review: rl, SourceName: sourceNameFromCandidate(c)})
	}

	out, semMerged := mergeSemanticRows(out)
	stats.SemanticMerged = semMerged
	stats.RowsStored = len(out)
	for _, r := range out {
		src := strings.TrimSpace(r.SourceName)
		if src == "" {
			src = sourceNameFromStaged(r.Staged)
		}
		sb := breakdown[src]
		if sb == nil {
			sb = &SourceBreakdown{SourceName: src}
			breakdown[src] = sb
		}
		switch r.Review.Action {
		case review.ActionContact:
			stats.ContactReady++
			sb.Qualified++
		case review.ActionResearchFirst:
			stats.ResearchFirst++
		}
	}
	var generatedTotal int
	for _, k := range []string{
		"google_discovery",
		"seed_discovery",
		"directory_discovery",
		"website_crawl_discovery",
		"job_signal_discovery",
		"mock_discovery",
	} {
		if sb, ok := breakdown[k]; ok {
			stats.SourceBreakdown = append(stats.SourceBreakdown, *sb)
			generatedTotal += sb.Generated
			delete(breakdown, k)
		}
	}
	for _, sb := range breakdown {
		stats.SourceBreakdown = append(stats.SourceBreakdown, *sb)
		generatedTotal += sb.Generated
	}
	stats.BreakdownGeneratedTotal = generatedTotal
	stats.BreakdownMatchesTotal = generatedTotal == stats.CandidatesFound
	return out, stats, nil
}

func sourceNameFromStaged(s domain.StagedOdooLead) string {
	switch s.Source {
	case domain.SourceGoogle:
		return "google_discovery"
	case domain.SourceCompanyWebsite:
		return "website_crawl_discovery"
	case domain.SourceJobPortal:
		return "job_signal_discovery"
	case domain.SourceApollo:
		return "apollo_enrichment"
	case domain.SourceLinkedIn:
		return "linkedin_signal"
	default:
		return "unknown_source"
	}
}

func sourceNameFromCandidate(c domain.RawCandidate) string {
	return c.PrimaryDiscoverySourceName()
}
