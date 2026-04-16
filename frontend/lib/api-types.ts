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

export interface SettingsResponse {
  discovery_sources: {
    google: boolean;
    seed: boolean;
    website_crawl: boolean;
    job_signal: boolean;
    apollo: boolean;
    linkedin: boolean;
  };
  icp: Record<string, unknown>;
  catalogs: {
    industries: { id: string; label: string; helper?: string }[];
    signals: unknown[];
    regions: unknown[];
    weights: string[];
  };
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
    discovery_mode?: string;
  };
}

export interface PipelineRunAPIResponse {
  run: {
    run_uuid: string;
    started_at?: string;
    finished_at?: string;
    status?: string;
    discovery_mode?: string;
  };
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
  };
  provider_statuses: unknown[];
  rows_persisted: number;
}
