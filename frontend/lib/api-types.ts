/**
 * Subset of backend DTOs (`internal/api/dto`) used by the web app.
 */

export interface ApiErrorBody {
  error: {
    code: string;
    message: string;
  };
}

/** Mirrors GET /api/v1/leads items[]. */
export interface Lead {
  id: number;
  company: string;
  industry: string;
  size?: string;
  icp_match: string;
  duplicate_status?: string;
  lead_status: string;
  confidence?: string;
  summary: string;
  reasons?: string[];
  source: string;
  created_at: string;
  website_domain?: string;
  linkedin_url?: string;
  country_region?: string;
  reason_for_fit?: string;
  why_now?: string;
  why_now_strength?: string;
  sales_angle?: string;
  priority_score: number;
  priority_level: string;
  data_completeness?: number;
  sales_status: string;
  employee_size?: string;
  accept_explanation?: string;
  missing_optional?: string[];
  source_ref?: string;
  sales_ready: boolean;
  action: string;
  official_domain?: string;
  source_trace?: string[];
  used_google?: boolean;
  used_apollo?: boolean;
  used_linkedin?: boolean;
  website_enrichment_selected_urls?: string;
  website_enrichment_summary?: string;
  website_enrichment_signals?: string;
  website_enrichment_status?: string;
  website_enriched_at?: string;
  original_discovery_source: string;
  enrichment_sources?: string[];
}

export interface PipelineSummaryNumbers {
  candidates_found: number;
  enriched: number;
  contact_ready: number;
  research_first: number;
  rejected: number;
  duplicates_removed: number;
  semantic_merged: number;
  rows_stored: number;
}

export interface LeadsListResponse {
  items: Lead[];
  pagination: { total: number; returned: number };
  summary: {
    last_run?: PipelineSummaryNumbers;
  };
  meta: {
    pipeline_has_run: boolean;
    total_in_db: number;
    industries: string[];
  };
  filter_echo: Record<string, string>;
}

export interface LeadDetailResponse {
  lead: Lead & Record<string, unknown>;
}

/** Mirrors `dto.DiscoverySourcesToggles` — JSON keys match Go struct tags. */
export interface DiscoverySourcesToggles {
  google: boolean;
  seed: boolean;
  website_crawl: boolean;
  job_signal: boolean;
  apollo: boolean;
  linkedin: boolean;
}

/** Mirrors `dto.ICPForm` — JSON keys match Go struct tags. */
export interface ICPForm {
  _v?: number;
  target_industries?: string[];
  region_focus?: string;
  signal_keys?: string[];
  excluded_industries?: string[];
  excluded_segments?: string[];
  apply_sub50_rule?: boolean | null;
  weight_industry?: string;
  weight_signal?: string;
  weight_size?: string;
  min_employees?: string;
  max_employees?: string;
  target_industry?: string;
  company_size?: string;
  country_region?: string;
  required_signal?: string;
  excluded_industry?: string;
}

export interface CatalogOption {
  id: string;
  label: string;
  helper?: string;
}

export interface SignalCatalogOption {
  id: string;
  label: string;
  helper?: string;
  keywords?: string[];
}

export interface SettingsCatalogs {
  industries: CatalogOption[];
  signals: SignalCatalogOption[];
  regions: CatalogOption[];
  weights: string[];
}

/** Mirrors `dto.DiscoveryIntegrationRow` — env/runtime readiness per discovery source. */
export interface DiscoveryIntegrationRow {
  key: string;
  available: boolean;
  enabled: boolean;
  requires_integration: boolean;
  /** Present only when `requires_integration` is true. */
  configured?: boolean;
  provider_name?: string;
  hint?: string;
}

export interface SettingsResponse {
  discovery_sources: DiscoverySourcesToggles;
  discovery_integrations?: DiscoveryIntegrationRow[];
  icp: ICPForm;
  catalogs: SettingsCatalogs;
}

/** Mirrors `dto.PutSettingsRequest` — PUT /api/v1/settings body. */
export interface PutSettingsRequest {
  discovery_sources: DiscoverySourcesToggles;
  icp: ICPForm;
}

export interface DebugResponse {
  no_runs_in_db: boolean;
  has_persisted_run: boolean;
  has_full_debug: boolean;
  summary: {
    pipeline_text?: string;
    has_run?: boolean;
    [key: string]: string | boolean | undefined;
  };
  run?: {
    run_uuid: string;
    started_at: string;
    status: string;
    run_outcome?: "success" | "partial_success" | "error";
    discovery_mode?: string;
  };
  run_outcome?: "success" | "partial_success" | "error";
  website_crawl_enabled?: boolean;
  website_crawl_configured?: boolean;
  website_crawl?: {
    enabled_in_settings: boolean;
    firecrawl_configured: boolean;
    status: string;
    reason_code?: string;
    reason_message?: string;
    skip_reason?: string;
    error_message?: string;
    pages_attempted: number;
    pages_succeeded: number;
  };
  website_crawl_metrics?: {
    upstream_candidate_pool: number;
    website_crawl_enrichment_attempted: number;
    website_crawl_enrichment_succeeded: number;
    true_website_crawl_discovered: number;
    true_website_crawl_discovery_supported: boolean;
    final_stored: number;
  };
  website_crawl_funnel?: {
    firecrawl_raw_candidates: number;
    firecrawl_after_domain_validation: number;
    firecrawl_after_dedupe: number;
    firecrawl_after_icp_filter: number;
    firecrawl_after_quality_gate: number;
    firecrawl_stored: number;
    drop_off_reasons: {
      dropped_no_valid_domain: number;
      dropped_duplicate: number;
      dropped_industry_mismatch: number;
      dropped_region_mismatch: number;
      dropped_employee_range_mismatch: number;
      dropped_low_confidence: number;
      dropped_low_signal_quality: number;
      dropped_quality_gate: number;
      dropped_other: number;
    };
  };
  provider_rows?: Array<{
    provider_name: string;
    state: string;
    skip_reason?: string;
    last_error?: string;
  }>;
  provider_details?: Array<{
    source_key: string;
    provider_name: string;
    configured: boolean;
    enabled_by_settings: boolean;
    status: string;
    reason_code?: string;
    reason_message?: string;
    skip_reason?: string;
    error_message?: string;
    details?: Record<string, unknown>;
    pages_attempted: number;
    pages_succeeded: number;
    candidates_total: number;
    candidates_success: number;
    candidates_skipped: number;
    candidates_failed: number;
    budget_limit_sec: number;
    budget_used_sec: number;
    budget_rows_skipped: number;
  }>;
  run_error_message?: string;
  breakdown_rows?: Array<{
    source_name: string;
    status: string;
    generated: number;
    kept: number;
    stored: number;
    skipped: number;
    failed: number;
    qualified: number;
    conversion: string;
    skip_reason?: string;
    last_error?: string;
  }>;
}

export interface PipelineRunAPIResponse {
  run: {
    run_uuid: string;
    started_at?: string;
    finished_at?: string;
    status?: string;
    run_outcome?: "success" | "partial_success" | "error";
    discovery_mode?: string;
  };
  run_outcome?: "success" | "partial_success" | "error";
  stats: {
    candidates_found: number;
    enriched: number;
    contact_ready: number;
    research_first: number;
    rejected: number;
    duplicates_removed: number;
    semantic_merged: number;
    rows_stored: number;
    integration_google_used?: boolean;
    integration_apollo_used?: boolean;
    integration_linkedin_used?: boolean;
    provider_statuses?: unknown[];
    source_breakdown?: unknown[];
    breakdown_generated_total?: number;
    breakdown_matches_total?: boolean;
    discovery_mode?: string;
    discovery_source?: string;
    run_outcome?: "success" | "partial_success" | "error";
    website_crawl_enabled?: boolean;
    website_crawl_configured?: boolean;
  };
  provider_statuses: unknown[];
  rows_persisted: number;
}
