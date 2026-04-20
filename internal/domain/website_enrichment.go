package domain

// WebsiteEnrichment captures website crawl output (Firecrawl or legacy HTTP).
type WebsiteEnrichment struct {
	SelectedURLs []string `json:"selected_urls,omitempty"`
	Summary      string   `json:"website_enrichment_summary,omitempty"`
	Signals      string   `json:"website_enrichment_signals,omitempty"`
	// PagesAttempted / PagesSucceeded are per-host enrichment counters for debug/ops.
	PagesAttempted int `json:"pages_attempted,omitempty"`
	PagesSucceeded int `json:"pages_succeeded,omitempty"`
	// Status is one of: success | failed | skipped | legacy_fallback
	Status string `json:"website_enrichment_status,omitempty"`
	// ReasonCode is a machine-readable explanation for skip/failure/degraded states.
	ReasonCode string `json:"website_enrichment_reason_code,omitempty"`
	// ReasonMessage is a short operator-facing explanation.
	ReasonMessage string `json:"website_enrichment_reason_message,omitempty"`
	// ErrorMessage is provider-specific raw context when available.
	ErrorMessage string `json:"website_enrichment_error_message,omitempty"`
	// EnrichedAt is RFC3339 UTC.
	EnrichedAt string `json:"website_enriched_at,omitempty"`
}
