CREATE INDEX IF NOT EXISTS idx_pipeline_runs_started_at ON pipeline_runs(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_status ON pipeline_runs(status);

CREATE INDEX IF NOT EXISTS idx_raw_candidates_run_id ON raw_candidates(pipeline_run_id);
CREATE INDEX IF NOT EXISTS idx_raw_candidates_source ON raw_candidates(source_name);
CREATE INDEX IF NOT EXISTS idx_raw_candidates_domain ON raw_candidates(domain_raw);

CREATE UNIQUE INDEX IF NOT EXISTS idx_companies_domain_unique
  ON companies(canonical_domain)
  WHERE canonical_domain IS NOT NULL AND canonical_domain != '';
CREATE INDEX IF NOT EXISTS idx_companies_name ON companies(normalized_name);
CREATE INDEX IF NOT EXISTS idx_companies_industry ON companies(industry);

CREATE INDEX IF NOT EXISTS idx_company_sources_company_run ON company_sources(company_id, pipeline_run_id);
CREATE INDEX IF NOT EXISTS idx_company_sources_source ON company_sources(source_name);

CREATE INDEX IF NOT EXISTS idx_company_signals_company_run ON company_signals(company_id, pipeline_run_id);
CREATE INDEX IF NOT EXISTS idx_company_signals_type ON company_signals(signal_type);

CREATE INDEX IF NOT EXISTS idx_company_scores_company_run ON company_scores(company_id, pipeline_run_id);
CREATE INDEX IF NOT EXISTS idx_company_scores_run_id ON company_scores(pipeline_run_id);
CREATE INDEX IF NOT EXISTS idx_company_scores_action ON company_scores(action);
CREATE INDEX IF NOT EXISTS idx_company_scores_score ON company_scores(score DESC);

-- Required common UI filters
CREATE INDEX IF NOT EXISTS idx_company_snapshots_pipeline_run_id ON company_snapshots(pipeline_run_id);
CREATE INDEX IF NOT EXISTS idx_company_snapshots_action ON company_snapshots(action);
CREATE INDEX IF NOT EXISTS idx_company_snapshots_priority_score ON company_snapshots(priority_score DESC);
CREATE INDEX IF NOT EXISTS idx_company_snapshots_industry ON company_snapshots(industry);
