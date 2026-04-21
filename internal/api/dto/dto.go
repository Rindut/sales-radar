// Package dto holds JSON shapes for /api/v1 responses and request bodies.
// Field names use snake_case to match existing pipeline/store JSON conventions.
package dto

import (
	"salesradar/internal/discovery"
	"salesradar/internal/pipeline"
)

// --- Common ---

// Pagination reflects current behavior: the list query returns the full filtered set (no DB offset/limit yet).
type Pagination struct {
	Total    int `json:"total"`
	Returned int `json:"returned"`
}

// ListMeta is included on the leads list response.
type ListMeta struct {
	PipelineHasRun bool     `json:"pipeline_has_run"`
	TotalInDB      int      `json:"total_in_db"`
	Industries     []string `json:"industries"`
}

// ListSummaryOptional captures optional last-run context (usually filled by the client after POST /pipeline/run).
type ListSummaryOptional struct {
	LastRun *PipelineSummaryNumbers `json:"last_run,omitempty"`
}

// PipelineSummaryNumbers mirrors stats the legacy UI showed in query strings after a run.
type PipelineSummaryNumbers struct {
	CandidatesFound   int `json:"candidates_found"`
	Enriched          int `json:"enriched"`
	ContactReady      int `json:"contact_ready"`
	ResearchFirst     int `json:"research_first"`
	Rejected          int `json:"rejected"`
	DuplicatesRemoved int `json:"duplicates_removed"`
	SemanticMerged    int `json:"semantic_merged"`
	RowsStored        int `json:"rows_stored"`
}

// LeadsListResponse is GET /api/v1/leads.
type LeadsListResponse struct {
	Items      []Lead              `json:"items"`
	Pagination Pagination          `json:"pagination"`
	Summary    ListSummaryOptional `json:"summary"`
	Meta       ListMeta            `json:"meta"`
	FilterEcho map[string]string   `json:"filter_echo"`
}

// LeadResponse is GET /api/v1/leads/{id}.
type LeadResponse struct {
	Lead Lead `json:"lead"`
}

// Lead is a JSON-safe view of store.Lead.
type Lead struct {
	ID                int64    `json:"id"`
	Company           string   `json:"company"`
	Industry          string   `json:"industry"`
	Size              string   `json:"size"`
	ICPMatch          string   `json:"icp_match"`
	DuplicateStatus   string   `json:"duplicate_status"`
	LeadStatus        string   `json:"lead_status"`
	Confidence        string   `json:"confidence"`
	Summary           string   `json:"summary"`
	Reasons           []string `json:"reasons"`
	Source            string   `json:"source"`
	CreatedAt         string   `json:"created_at"`
	WebsiteDomain     string   `json:"website_domain"`
	LinkedInURL       string   `json:"linkedin_url"`
	CountryRegion     string   `json:"country_region"`
	ReasonForFit      string   `json:"reason_for_fit"`
	WhyNow            string   `json:"why_now"`
	WhyNowStrength    string   `json:"why_now_strength"`
	SalesAngle        string   `json:"sales_angle"`
	PriorityScore     int      `json:"priority_score"`
	PriorityLevel     string   `json:"priority_level"`
	DataCompleteness  int      `json:"data_completeness"`
	SalesStatus       string   `json:"sales_status"`
	EmployeeSize      string   `json:"employee_size"`
	AcceptExplanation string   `json:"accept_explanation"`
	MissingOptional   []string `json:"missing_optional"`
	SourceRef         string   `json:"source_ref"`
	SalesReady        bool     `json:"sales_ready"`
	Action            string   `json:"action"`
	OfficialDomain    string   `json:"official_domain"`
	SourceTrace       []string `json:"source_trace"`
	UsedGoogle        bool     `json:"used_google"`
	UsedApollo        bool     `json:"used_apollo"`
	UsedLinkedIn      bool     `json:"used_linkedin"`
	// Website enrichment (Firecrawl / legacy crawl); optional on list/detail.
	WebsiteEnrichmentSelectedURLs string   `json:"website_enrichment_selected_urls,omitempty"`
	WebsiteEnrichmentSummary      string   `json:"website_enrichment_summary,omitempty"`
	WebsiteEnrichmentSignals      string   `json:"website_enrichment_signals,omitempty"`
	WebsiteEnrichmentStatus       string   `json:"website_enrichment_status,omitempty"`
	WebsiteEnrichedAt             string   `json:"website_enriched_at,omitempty"`
	OriginalDiscoverySource       string   `json:"original_discovery_source"`
	EnrichmentSources             []string `json:"enrichment_sources,omitempty"`
}

// --- Settings ---

// DiscoverySourcesToggles is persisted discovery source switches.
type DiscoverySourcesToggles struct {
	Google       bool `json:"google"`
	Seed         bool `json:"seed"`
	WebsiteCrawl bool `json:"website_crawl"`
	JobSignal    bool `json:"job_signal"`
	Apollo       bool `json:"apollo"`
	LinkedIn     bool `json:"linkedin"`
}

// ICPForm is persisted ICP settings (mirrors store.ICPFormSettings JSON tags).
type ICPForm struct {
	Version            int      `json:"_v,omitempty"`
	TargetIndustries   []string `json:"target_industries,omitempty"`
	RegionFocus        string   `json:"region_focus,omitempty"`
	SignalKeys         []string `json:"signal_keys,omitempty"`
	ExcludedIndustries []string `json:"excluded_industries,omitempty"`
	ExcludedSegments   []string `json:"excluded_segments,omitempty"`
	ApplySub50         *bool    `json:"apply_sub50_rule,omitempty"`
	WeightIndustry     string   `json:"weight_industry,omitempty"`
	WeightSignal       string   `json:"weight_signal,omitempty"`
	WeightSize         string   `json:"weight_size,omitempty"`
	MinEmployees       string   `json:"min_employees,omitempty"`
	MaxEmployees       string   `json:"max_employees,omitempty"`
	TargetIndustry     string   `json:"target_industry,omitempty"`
	CompanySize        string   `json:"company_size,omitempty"`
	CountryRegion      string   `json:"country_region,omitempty"`
	RequiredSignal     string   `json:"required_signal,omitempty"`
	ExcludedIndustry   string   `json:"excluded_industry,omitempty"`
}

// CatalogOption is a selectable catalog row (industry, signal, region).
type CatalogOption struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Helper string `json:"helper,omitempty"`
}

// SignalCatalogOption extends catalog signal with keywords.
type SignalCatalogOption struct {
	ID       string   `json:"id"`
	Label    string   `json:"label"`
	Helper   string   `json:"helper,omitempty"`
	Keywords []string `json:"keywords,omitempty"`
}

// SettingsCatalogs is static catalog data for the settings UI.
type SettingsCatalogs struct {
	Industries []CatalogOption       `json:"industries"`
	Signals    []SignalCatalogOption `json:"signals"`
	Regions    []CatalogOption       `json:"regions"`
	Weights    []string              `json:"weights"`
}

// DiscoveryIntegrationRow describes external integration readiness for one discovery source.
type DiscoveryIntegrationRow struct {
	Key                 string `json:"key"`
	Available           bool   `json:"available"`
	Enabled             bool   `json:"enabled"`
	RequiresIntegration bool   `json:"requires_integration"`
	// Configured is only set when RequiresIntegration is true (pointer so JSON can emit false).
	Configured *bool `json:"configured,omitempty"`
	// ProviderName is optional UI hint (e.g. Firecrawl for website crawl).
	ProviderName string `json:"provider_name,omitempty"`
	Hint         string `json:"hint,omitempty"`
}

// SettingsResponse is GET /api/v1/settings.
type SettingsResponse struct {
	DiscoverySources      DiscoverySourcesToggles   `json:"discovery_sources"`
	DiscoveryIntegrations []DiscoveryIntegrationRow `json:"discovery_integrations"`
	ICP                   ICPForm                   `json:"icp"`
	Catalogs              SettingsCatalogs          `json:"catalogs"`
}

// PutSettingsRequest is PUT /api/v1/settings (full replacement of discovery + ICP).
type PutSettingsRequest struct {
	DiscoverySources DiscoverySourcesToggles `json:"discovery_sources"`
	ICP              ICPForm                 `json:"icp"`
}

// --- Pipeline ---

// PipelineRunResponse is POST /api/v1/pipeline/run.
// Provider statuses are included both in stats.provider_statuses and top-level for clients that expect a flat contract.
type PipelineRunResponse struct {
	Run              PipelineRunInfo            `json:"run"`
	Stats            pipeline.RunStats          `json:"stats"`
	ProviderStatuses []discovery.ProviderStatus `json:"provider_statuses"`
	RowsPersisted    int                        `json:"rows_persisted"`
	RunOutcome       pipeline.RunOutcome        `json:"run_outcome"`
}

// PipelineRunInfo is minimal persisted-run metadata after a successful replace.
type PipelineRunInfo struct {
	RunUUID       string `json:"run_uuid"`
	StartedAt     string `json:"started_at,omitempty"`
	FinishedAt    string `json:"finished_at,omitempty"`
	Status        string `json:"status,omitempty"`
	RunOutcome    string `json:"run_outcome,omitempty"`
	DiscoveryMode string `json:"discovery_mode,omitempty"`
}

// --- Debug ---

// DebugRunMeta is the latest pipeline_runs row (ops view).
type DebugRunMeta struct {
	RunID         int64  `json:"run_id"`
	RunUUID       string `json:"run_uuid"`
	StartedAt     string `json:"started_at"`
	FinishedAt    string `json:"finished_at,omitempty"`
	Status        string `json:"status"`
	RunOutcome    string `json:"run_outcome"`
	DiscoveryMode string `json:"discovery_mode"`
	HasDebugJSON  bool   `json:"has_debug_json"`
}

type DebugProviderDetail struct {
	SourceKey         string         `json:"source_key"`
	ProviderName      string         `json:"provider_name"`
	Configured        bool           `json:"configured"`
	EnabledBySettings bool           `json:"enabled_by_settings"`
	Status            string         `json:"status"`
	ReasonCode        string         `json:"reason_code,omitempty"`
	ReasonMessage     string         `json:"reason_message,omitempty"`
	SkipReason        string         `json:"skip_reason,omitempty"`
	ErrorMessage      string         `json:"error_message,omitempty"`
	Details           map[string]any `json:"details,omitempty"`
	PagesAttempted    int            `json:"pages_attempted"`
	PagesSucceeded    int            `json:"pages_succeeded"`
	CandidatesTotal   int            `json:"candidates_total"`
	CandidatesSuccess int            `json:"candidates_success"`
	CandidatesSkipped int            `json:"candidates_skipped"`
	CandidatesFailed  int            `json:"candidates_failed"`
	BudgetLimitSec    int            `json:"budget_limit_sec"`
	BudgetUsedSec     int            `json:"budget_used_sec"`
	BudgetRowsSkipped int            `json:"budget_rows_skipped"`
}

type DebugWebsiteCrawlSummary struct {
	EnabledInSettings   bool   `json:"enabled_in_settings"`
	FirecrawlConfigured bool   `json:"firecrawl_configured"`
	Status              string `json:"status"`
	ReasonCode          string `json:"reason_code,omitempty"`
	ReasonMessage       string `json:"reason_message,omitempty"`
	SkipReason          string `json:"skip_reason,omitempty"`
	ErrorMessage        string `json:"error_message,omitempty"`
	PagesAttempted      int    `json:"pages_attempted"`
	PagesSucceeded      int    `json:"pages_succeeded"`
}

type DebugWebsiteCrawlDropOffReasons struct {
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

type DebugWebsiteCrawlFunnel struct {
	RawCandidates         int                             `json:"firecrawl_raw_candidates"`
	AfterDomainValidation int                             `json:"firecrawl_after_domain_validation"`
	AfterDedupe           int                             `json:"firecrawl_after_dedupe"`
	AfterICPFilter        int                             `json:"firecrawl_after_icp_filter"`
	AfterQualityGate      int                             `json:"firecrawl_after_quality_gate"`
	Stored                int                             `json:"firecrawl_stored"`
	DropOffReasons        DebugWebsiteCrawlDropOffReasons `json:"drop_off_reasons"`
}

type DebugWebsiteCrawlMetrics struct {
	UpstreamCandidatePool           int  `json:"upstream_candidate_pool"`
	WebsiteCrawlEnrichmentAttempted int  `json:"website_crawl_enrichment_attempted"`
	WebsiteCrawlEnrichmentSucceeded int  `json:"website_crawl_enrichment_succeeded"`
	TrueWebsiteCrawlDiscovered      int  `json:"true_website_crawl_discovered"`
	TrueDiscoverySupported          bool `json:"true_website_crawl_discovery_supported"`
	FinalStored                     int  `json:"final_stored"`
}

// DebugIntegrationRow is a host/integration summary row.
type DebugIntegrationRow struct {
	Host    string `json:"host"`
	Role    string `json:"role"`
	Config  string `json:"config"`
	LastRun string `json:"last_run"`
}

// DebugBreakdownRow mirrors the HTML debug table.
type DebugBreakdownRow struct {
	SourceName       string  `json:"source_name"`
	Status           string  `json:"status"`
	Generated        int     `json:"generated"`
	Kept             int     `json:"kept"`
	Stored           int     `json:"stored"`
	Skipped          int     `json:"skipped"`
	Failed           int     `json:"failed"`
	Qualified        int     `json:"qualified"`
	Conversion       string  `json:"conversion"`
	ConversionPct    float64 `json:"conversion_pct"`
	SkipReason       string  `json:"skip_reason"`
	LastError        string  `json:"last_error"`
	IsError          bool    `json:"is_error"`
	IsHighConversion bool    `json:"is_high_conversion"`
}

// DebugSummary holds human-readable pipeline summary strings when debug JSON exists.
type DebugSummary struct {
	HasRun        bool   `json:"has_run"`
	TotalLeads    string `json:"total_leads"`
	ContactReady  string `json:"contact_ready"`
	PendingReview string `json:"pending_review"`
	Candidates    string `json:"candidates"`
	Enriched      string `json:"enriched"`
	Rejected      string `json:"rejected"`
	Duplicates    string `json:"duplicates"`
	Merged        string `json:"merged"`
	PipelineText  string `json:"pipeline_text"`
}

// DebugResponse is GET /api/v1/debug.
type DebugResponse struct {
	Run                    *DebugRunMeta              `json:"run,omitempty"`
	StatsDecodeError       string                     `json:"stats_decode_error,omitempty"`
	NoRunsInDB             bool                       `json:"no_runs_in_db"`
	HasPersistedRun        bool                       `json:"has_persisted_run"`
	HasFullDebug           bool                       `json:"has_full_debug"`
	Summary                DebugSummary               `json:"summary"`
	ProviderRows           []discovery.ProviderStatus `json:"provider_rows"`
	BreakdownRows          []DebugBreakdownRow        `json:"breakdown_rows"`
	BreakdownTotal         string                     `json:"breakdown_total"`
	BreakdownOK            bool                       `json:"breakdown_ok"`
	DiscoveryMode          string                     `json:"discovery_mode"`
	DiscoverySource        string                     `json:"discovery_source"`
	IntegrationRows        []DebugIntegrationRow      `json:"integration_rows"`
	ProviderDetails        []DebugProviderDetail      `json:"provider_details"`
	WebsiteCrawl           DebugWebsiteCrawlSummary   `json:"website_crawl"`
	WebsiteCrawlMetrics    DebugWebsiteCrawlMetrics   `json:"website_crawl_metrics"`
	WebsiteCrawlFunnel     DebugWebsiteCrawlFunnel    `json:"website_crawl_funnel"`
	RunOutcome             pipeline.RunOutcome        `json:"run_outcome"`
	WebsiteCrawlEnabled    bool                       `json:"website_crawl_enabled"`
	WebsiteCrawlConfigured bool                       `json:"website_crawl_configured"`
	RunErrorMessage        string                     `json:"run_error_message,omitempty"`
}
