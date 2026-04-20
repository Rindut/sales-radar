package pipeline

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"salesradar/internal/deduplication"
	"salesradar/internal/discovery"
	"salesradar/internal/domain"
	"salesradar/internal/enrichment"
	"salesradar/internal/extraction"
	"salesradar/internal/failpoint"
	"salesradar/internal/icp"
	"salesradar/internal/odoo"
	"salesradar/internal/quality"
	"salesradar/internal/review"
	"salesradar/internal/status"
)

var ErrCorePipelineFailpoint = errors.New("core pipeline error forced by failpoint")

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
	IntegrationGoogleUsed   bool                       `json:"integration_google_used"`
	IntegrationApolloUsed   bool                       `json:"integration_apollo_used"`
	IntegrationLinkedInUsed bool                       `json:"integration_linkedin_used"`
	ProviderStatuses        []discovery.ProviderStatus `json:"provider_statuses"`
	DiscoveryMode           string                     `json:"discovery_mode"`
	DiscoverySource         string                     `json:"discovery_source"`
	SourceBreakdown         []SourceBreakdown          `json:"source_breakdown"`
	BreakdownGeneratedTotal int                        `json:"breakdown_generated_total"`
	BreakdownMatchesTotal   bool                       `json:"breakdown_matches_total"`
	RunOutcome              RunOutcome                 `json:"run_outcome"`
	WebsiteCrawlEnabled     bool                       `json:"website_crawl_enabled"`
	WebsiteCrawlConfigured  bool                       `json:"website_crawl_configured"`
	WebsiteCrawlFunnel      WebsiteCrawlFunnel         `json:"website_crawl_funnel"`
}

type WebsiteCrawlDropOffReasons struct {
	DroppedNoValidDomain         int `json:"dropped_no_valid_domain"`
	DroppedDuplicate             int `json:"dropped_duplicate"`
	DroppedIndustryMismatch      int `json:"dropped_industry_mismatch"`
	DroppedRegionMismatch        int `json:"dropped_region_mismatch"`
	DroppedEmployeeRangeMismatch int `json:"dropped_employee_range_mismatch"`
	DroppedLowConfidence         int `json:"dropped_low_confidence"`
	DroppedLowSignalQuality      int `json:"dropped_low_signal_quality"`
	DroppedQualityGate           int `json:"dropped_quality_gate"`
	DroppedOther                 int `json:"dropped_other"`
}

type WebsiteCrawlFunnel struct {
	RawCandidates         int                        `json:"firecrawl_raw_candidates"`
	AfterDomainValidation int                        `json:"firecrawl_after_domain_validation"`
	AfterDedupe           int                        `json:"firecrawl_after_dedupe"`
	AfterICPFilter        int                        `json:"firecrawl_after_icp_filter"`
	AfterQualityGate      int                        `json:"firecrawl_after_quality_gate"`
	Stored                int                        `json:"firecrawl_stored"`
	DropOffReasons        WebsiteCrawlDropOffReasons `json:"drop_off_reasons"`
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
	Staged     domain.StagedOdooLead
	Review     review.ReviewLead
	SourceName string
}

// RunWithQualityGate runs the pipeline, applies storage rules, and returns rows to persist plus stats.
func RunWithQualityGate(ctx context.Context, params domain.RunParams) ([]PreparedRow, RunStats, error) {
	if failpoint.CorePipelineError() {
		slog.Warn("pipeline: failpoint forced core pipeline error")
		return nil, RunStats{}, ErrCorePipelineFailpoint
	}
	slog.Info("pipeline: discovery starting")
	disRes, err := discovery.DiscoverWithStatus(ctx, params)
	if err != nil {
		slog.Warn("pipeline: discovery failed", "err", err)
		return nil, RunStats{}, err
	}
	raw := disRes.Candidates
	slog.Info("pipeline: discovery complete", "raw_candidates", len(raw))

	store := noopDedupStore{}
	client := noopOdooClient{}

	var stats RunStats
	stats.ProviderStatuses = append([]discovery.ProviderStatus(nil), disRes.Providers...)
	stats.DiscoveryMode = disRes.Mode
	stats.DiscoverySource = disRes.Source
	stats.CandidatesFound = len(raw)
	toggles := domain.SourceTogglesOrDefault(params.SourceToggles)
	stats.WebsiteCrawlEnabled = toggles.WebsiteCrawl
	for i := range stats.ProviderStatuses {
		if stats.ProviderStatuses[i].ProviderName == "website_crawl_discovery" {
			if stats.ProviderStatuses[i].Configured != nil {
				stats.WebsiteCrawlConfigured = *stats.ProviderStatuses[i].Configured
			}
			break
		}
	}
	stats.WebsiteCrawlFunnel = websiteFunnelFromProviderStats(stats.ProviderStatuses)
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

	slog.Info("pipeline: extract/icp/dedup/status loop starting", "rows", len(raw))
	for _, c := range raw {
		src := sourceNameFromCandidate(c)
		isWebsite := src == "website_crawl_discovery"
		ext, err := extraction.Extract(ctx, c)
		if err != nil {
			return nil, RunStats{}, err
		}
		enrichment.Apply(ctx, ext)
		icpLead, err := icp.Evaluate(ctx, ext, icpCfg)
		if err != nil {
			return nil, RunStats{}, err
		}
		if isWebsite {
			if icpLead.ICPMatch != domain.ICPNo {
				stats.WebsiteCrawlFunnel.AfterICPFilter++
			} else {
				accumulateWebsiteICPDropReasons(&stats.WebsiteCrawlFunnel.DropOffReasons, icpLead.ICPReason)
			}
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
			breakdown[src].Rejected++
			if isWebsite {
				stats.WebsiteCrawlFunnel.DropOffReasons.DroppedDuplicate++
			}
			continue
		}
		if isWebsite {
			// Candidate survives duplicate removal stage.
			stats.WebsiteCrawlFunnel.AfterDedupe++
		}
		if staged.ICPMatch == domain.ICPNo {
			stats.Rejected++
			breakdown[src].Rejected++
			continue
		}

		rl := review.BuildReviewLead(*staged)
		if review.HasEnrichmentSignals(rl) {
			stats.Enriched++
		}

		if !quality.PassesStorageGate(rl) {
			stats.Rejected++
			breakdown[src].Rejected++
			if isWebsite {
				stats.WebsiteCrawlFunnel.DropOffReasons.DroppedQualityGate++
				accumulateWebsiteQualityGateDropReasons(&stats.WebsiteCrawlFunnel.DropOffReasons, rl)
			}
			continue
		}
		if isWebsite {
			stats.WebsiteCrawlFunnel.AfterQualityGate++
		}

		rl = review.ApplySalesStatusAndCopy(*staged, rl)
		if rl.Action == review.ActionIgnore {
			stats.Rejected++
			breakdown[src].Rejected++
			if isWebsite {
				stats.WebsiteCrawlFunnel.DropOffReasons.DroppedQualityGate++
				if strings.EqualFold(rl.Confidence, review.ConfidenceLow) {
					stats.WebsiteCrawlFunnel.DropOffReasons.DroppedLowConfidence++
				}
			}
			continue
		}
		breakdown[src].Kept++
		out = append(out, PreparedRow{Staged: *staged, Review: rl, SourceName: src})
		if isWebsite {
			stats.WebsiteCrawlFunnel.Stored++
		}
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
	stats.RunOutcome = decideRunOutcome(stats, toggles)
	slog.Info("pipeline: finished", "rows_stored", stats.RowsStored, "candidates", stats.CandidatesFound, "rejected", stats.Rejected)
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

func websiteFunnelFromProviderStats(statuses []discovery.ProviderStatus) WebsiteCrawlFunnel {
	var f WebsiteCrawlFunnel
	for _, st := range statuses {
		if st.ProviderName != "website_crawl_discovery" {
			continue
		}
		if v, ok := intFromDetails(st.Details, "pool"); ok {
			f.RawCandidates = v
		}
		if v, ok := intFromDetails(st.Details, "with_domain"); ok {
			f.AfterDomainValidation = v
		}
		f.DropOffReasons.DroppedNoValidDomain = intFromDetailsDefault(st.Details, "no_domain", 0)
		if f.RawCandidates == 0 {
			f.RawCandidates = st.CandidatesTotal
		}
		if f.AfterDomainValidation == 0 {
			f.AfterDomainValidation = f.RawCandidates - f.DropOffReasons.DroppedNoValidDomain
			if f.AfterDomainValidation < 0 {
				f.AfterDomainValidation = 0
			}
		}
		break
	}
	return f
}

func intFromDetails(details map[string]any, key string) (int, bool) {
	if details == nil {
		return 0, false
	}
	v, ok := details[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case int:
		return n, true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}

func intFromDetailsDefault(details map[string]any, key string, d int) int {
	if v, ok := intFromDetails(details, key); ok {
		return v
	}
	return d
}

func accumulateWebsiteICPDropReasons(drop *WebsiteCrawlDropOffReasons, reasons []string) {
	if drop == nil {
		return
	}
	if len(reasons) == 0 {
		drop.DroppedOther++
		return
	}
	matched := false
	for _, r := range reasons {
		t := strings.ToLower(strings.TrimSpace(r))
		switch {
		case strings.Contains(t, "non-target industry"):
			drop.DroppedIndustryMismatch++
			matched = true
		case strings.Contains(t, "region"):
			drop.DroppedRegionMismatch++
			matched = true
		case strings.Contains(t, "below minimum company size"),
			strings.Contains(t, "above maximum company size"),
			strings.Contains(t, "below 50 employees"),
			strings.Contains(t, "below typical company size"):
			drop.DroppedEmployeeRangeMismatch++
			matched = true
		}
	}
	if !matched {
		drop.DroppedOther++
	}
}

func accumulateWebsiteQualityGateDropReasons(drop *WebsiteCrawlDropOffReasons, rl review.ReviewLead) {
	if drop == nil {
		return
	}
	missing := quality.RequiredMissing(rl)
	matched := false
	for _, m := range missing {
		switch strings.TrimSpace(m) {
		case "employee_size_or_website_or_reason_for_fit":
			drop.DroppedLowSignalQuality++
			matched = true
		}
	}
	if strings.EqualFold(rl.Confidence, review.ConfidenceLow) {
		drop.DroppedLowConfidence++
		matched = true
	}
	if !matched {
		drop.DroppedOther++
	}
}
