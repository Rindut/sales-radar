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
	PipelineHasRun bool   `json:"pipeline_has_run"`
	TotalInDB      int    `json:"total_in_db"`
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
	Items       []Lead            `json:"items"`
	Pagination  Pagination        `json:"pagination"`
	Summary     ListSummaryOptional `json:"summary"`
	Meta        ListMeta          `json:"meta"`
	FilterEcho  map[string]string `json:"filter_echo"`
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
	Version             int      `json:"_v,omitempty"`
	TargetIndustries    []string `json:"target_industries,omitempty"`
	RegionFocus         string   `json:"region_focus,omitempty"`
	SignalKeys          []string `json:"signal_keys,omitempty"`
	ExcludedIndustries  []string `json:"excluded_industries,omitempty"`
	ExcludedSegments    []string `json:"excluded_segments,omitempty"`
	ApplySub50          *bool    `json:"apply_sub50_rule,omitempty"`
	WeightIndustry      string   `json:"weight_industry,omitempty"`
	WeightSignal        string   `json:"weight_signal,omitempty"`
	WeightSize          string   `json:"weight_size,omitempty"`
	MinEmployees        string   `json:"min_employees,omitempty"`
	MaxEmployees        string   `json:"max_employees,omitempty"`
	TargetIndustry      string   `json:"target_industry,omitempty"`
	CompanySize         string   `json:"company_size,omitempty"`
	CountryRegion       string   `json:"country_region,omitempty"`
	RequiredSignal      string   `json:"required_signal,omitempty"`
	ExcludedIndustry    string   `json:"excluded_industry,omitempty"`
}

// CatalogOption is a selectable catalog row (industry, signal, region).
type CatalogOption struct {
	ID     string   `json:"id"`
	Label  string   `json:"label"`
	Helper string   `json:"helper,omitempty"`
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

// SettingsResponse is GET /api/v1/settings.
type SettingsResponse struct {
	DiscoverySources DiscoverySourcesToggles `json:"discovery_sources"`
	ICP              ICPForm                 `json:"icp"`
	Catalogs         SettingsCatalogs        `json:"catalogs"`
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
	Run                PipelineRunInfo           `json:"run"`
	Stats              pipeline.RunStats         `json:"stats"`
	ProviderStatuses   []discovery.ProviderStatus `json:"provider_statuses"`
	RowsPersisted      int                       `json:"rows_persisted"`
}

// PipelineRunInfo is minimal persisted-run metadata after a successful replace.
type PipelineRunInfo struct {
	RunUUID       string `json:"run_uuid"`
	StartedAt     string `json:"started_at,omitempty"`
	FinishedAt    string `json:"finished_at,omitempty"`
	Status        string `json:"status,omitempty"`
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
	DiscoveryMode string `json:"discovery_mode"`
	HasDebugJSON  bool   `json:"has_debug_json"`
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
	Run              *DebugRunMeta           `json:"run,omitempty"`
	StatsDecodeError string                  `json:"stats_decode_error,omitempty"`
	NoRunsInDB       bool                    `json:"no_runs_in_db"`
	HasPersistedRun  bool                    `json:"has_persisted_run"`
	HasFullDebug     bool                    `json:"has_full_debug"`
	Summary          DebugSummary            `json:"summary"`
	ProviderRows     []discovery.ProviderStatus `json:"provider_rows"`
	BreakdownRows    []DebugBreakdownRow     `json:"breakdown_rows"`
	BreakdownTotal   string                  `json:"breakdown_total"`
	BreakdownOK      bool                    `json:"breakdown_ok"`
	DiscoveryMode    string                  `json:"discovery_mode"`
	DiscoverySource  string                  `json:"discovery_source"`
	IntegrationRows  []DebugIntegrationRow   `json:"integration_rows"`
}
