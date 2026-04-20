package domain

import "strings"

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

// PrimaryDiscoverySourceName is the canonical breakdown key for pipeline/UI (first source_trace tag, else Source).
func (c RawCandidate) PrimaryDiscoverySourceName() string {
	if len(c.ProspectTrace.SourceTrace) > 0 {
		s := c.ProspectTrace.SourceTrace[0]
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	switch c.Source {
	case SourceGoogle:
		return "google_discovery"
	case SourceCompanyWebsite:
		return "website_crawl_discovery"
	case SourceJobPortal:
		return "job_signal_discovery"
	case SourceApollo:
		return "apollo_enrichment"
	case SourceLinkedIn:
		return "linkedin_signal"
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
