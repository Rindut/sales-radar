package domain

import "strings"

const (
	TraceGoogleDiscovery     = "google_discovery"
	TraceSeedDiscovery       = "seed_discovery"
	TraceDirectoryDiscovery  = "directory_discovery"
	TraceWebsiteDiscovery    = "website_crawl_discovery"
	TraceWebsiteEnrichment   = "website_crawl_enrichment"
	TraceJobSignalDiscovery  = "job_signal_discovery"
	TraceMockDiscovery       = "mock_discovery"
	TraceApolloDiscovery     = "apollo_discovery"
	TraceApolloEnrichment    = "apollo_enrichment"
	TraceLinkedInSignal      = "linkedin_signal"
	TraceLinkedInValidation  = "linkedin_validation"
	TraceCompanyWebsiteCheck = "company_website_check"
)

// RunParams is the input contract for a discovery run.
type RunParams struct {
	MaxLeadsThisRun int
	SourceAllowlist []Source
	// SourceToggles is optional; nil means all discovery sources enabled per DefaultDiscoverySourceToggles.
	SourceToggles *DiscoverySourceToggles `json:"-"`
	// ICPRuntime is optional; nil means DefaultICPRuntimeSettings() at evaluate time.
	ICPRuntime *ICPRuntimeSettings `json:"-"`
}

// ProspectTrace records which external systems were used and an ordered list of steps applied.
// Keep values human-readable (product v2), e.g.:
// - "google_discovery"
// - "google_domain_extraction"
// - "apollo_enrichment"
// - "linkedin_validation"
// - "company_website_check"
type ProspectTrace struct {
	UsedGoogle   bool
	UsedApollo   bool
	UsedLinkedIn bool
	SourceTrace  []string
}

// RawCandidate is discovery output before structured extraction.
type RawCandidate struct {
	DiscoveryID         string
	Source              Source
	SourceRef           string
	UnstructuredContext string
	// OfficialDomain is the registrable company website host (never google.com / linkedin.com / apollo.io).
	OfficialDomain string
	ProspectTrace  ProspectTrace
	// EnrichedLinkedInURL is a company page URL from Apollo (or similar), not a search-results placeholder.
	EnrichedLinkedInURL string
	// WebsiteEnrichment is set when website crawl / Firecrawl produced structured context (optional).
	WebsiteEnrichment *WebsiteEnrichment
}

// NormalizedLead is canonical discovery output used by downstream modules.
type NormalizedLead struct {
	CompanyName string
	Domain      string
	Industry    string
}

// PrimaryDiscoverySourceName returns the original discovery source for pipeline/UI.
// It intentionally ignores later enrichment steps so they do not masquerade as discovery.
func (c RawCandidate) PrimaryDiscoverySourceName() string {
	return PrimaryDiscoverySourceNameFromTrace(c.ProspectTrace.SourceTrace, c.Source)
}

// PrimaryDiscoverySourceNameFromTrace returns the original discovery attribution from trace + fallback source.
// If website crawl appears alongside another discovery source, the non-crawl discovery source wins because
// website crawl currently acts primarily as enrichment on top of an upstream pool.
func PrimaryDiscoverySourceNameFromTrace(trace []string, fallback Source) string {
	discovery := uniqueDiscoverySources(trace)
	for _, src := range discovery {
		if src != TraceWebsiteDiscovery {
			return src
		}
	}
	if len(discovery) > 0 {
		return discovery[0]
	}
	return fallbackSourceName(fallback)
}

// EnrichmentSourceNamesFromTrace returns enrichment steps separately from discovery attribution.
func EnrichmentSourceNamesFromTrace(trace []string, hasWebsiteEnrichment bool) []string {
	out := make([]string, 0, 3)
	seen := map[string]struct{}{}
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	if hasWebsiteEnrichment {
		add(TraceWebsiteEnrichment)
	}
	for _, raw := range trace {
		switch strings.TrimSpace(raw) {
		case TraceWebsiteEnrichment, TraceApolloEnrichment, TraceLinkedInSignal, TraceLinkedInValidation, TraceCompanyWebsiteCheck:
			add(strings.TrimSpace(raw))
		}
	}
	return out
}

func uniqueDiscoverySources(trace []string) []string {
	out := make([]string, 0, 4)
	seen := map[string]struct{}{}
	for _, raw := range trace {
		src := strings.TrimSpace(raw)
		if !isDiscoveryTrace(src) {
			continue
		}
		if _, ok := seen[src]; ok {
			continue
		}
		seen[src] = struct{}{}
		out = append(out, src)
	}
	return out
}

func isDiscoveryTrace(src string) bool {
	switch strings.TrimSpace(src) {
	case TraceGoogleDiscovery, TraceSeedDiscovery, TraceDirectoryDiscovery, TraceWebsiteDiscovery, TraceJobSignalDiscovery, TraceMockDiscovery, TraceApolloDiscovery:
		return true
	default:
		return false
	}
}

func fallbackSourceName(src Source) string {
	switch src {
	case SourceGoogle:
		return TraceGoogleDiscovery
	case SourceCompanyWebsite:
		return TraceWebsiteDiscovery
	case SourceJobPortal:
		return TraceJobSignalDiscovery
	case SourceApollo:
		return TraceApolloDiscovery
	case SourceLinkedIn:
		return TraceLinkedInSignal
	default:
		return "unknown_source"
	}
}

// ExtractedLead is structured company data after extraction.
type ExtractedLead struct {
	DiscoveryID          string
	Source               Source
	SourceRef            string
	CompanyName          *string
	Industry             *string
	StrongClassification bool
	ICPIndustryBucket    ICPIndustryBucket
	CompanySizeEstimated *string
	Location             *string
	AISummaryShort       *string
	ExtractionNotes      *string
	// UnstructuredContext is the raw discovery blurb (used by ICP for LXP/industry signals).
	UnstructuredContext string
	OfficialDomain      string
	ProspectTrace       ProspectTrace
	EnrichedLinkedInURL string
	WebsiteEnrichment   *WebsiteEnrichment
}

// ICPLead adds ICP classification to an extracted lead.
type ICPLead struct {
	ExtractedLead
	ICPMatch  ICPMatch
	ICPReason []string
	// ICPScore is 0–100 from the scoring engine (industry, size, signals, domain).
	ICPScore int
	// ScoreAction aligns with PRD: Contact | Research | Reject (UI maps Reject → Ignore).
	ScoreAction ScoreAction
}

// DedupedLead adds duplicate classification.
type DedupedLead struct {
	ICPLead
	DuplicateStatus DuplicateStatus
}

// StagedOdooLead is ready for CRM push after status assignment.
type StagedOdooLead struct {
	DedupedLead
	Status      LeadStatus
	Explanation []string // human-readable reasons (ICP + dedup + status); max ~1 line each
}

// OdooPushResult is the outcome of a push attempt (retries tracked outside or here).
type OdooPushResult struct {
	OdooRecordID *string
	Outcome      OdooPushOutcome
	ErrorCode    *string
	Attempts     int
}

// DiscardedLead captures a non-pushed record for internal audit.
type DiscardedLead struct {
	Snapshot       ICPLead
	DiscardCodes   []DiscardCode
	InternalReason string
}

// DeferredLead holds overflow for a later run (same contract as staged).
type DeferredLead struct {
	Staged StagedOdooLead
	Reason string
}
