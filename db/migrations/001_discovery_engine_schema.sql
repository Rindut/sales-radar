CREATE TABLE IF NOT EXISTS pipeline_runs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  run_uuid TEXT NOT NULL UNIQUE,
  triggered_by TEXT,
  discovery_mode TEXT NOT NULL,
  scoring_version TEXT NOT NULL,
  started_at TEXT NOT NULL,
  finished_at TEXT,
  status TEXT NOT NULL DEFAULT 'running',
  total_raw_candidates INTEGER NOT NULL DEFAULT 0,
  total_companies INTEGER NOT NULL DEFAULT 0,
  total_snapshots INTEGER NOT NULL DEFAULT 0,
  error_message TEXT
);

CREATE TABLE IF NOT EXISTS raw_candidates (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  pipeline_run_id INTEGER NOT NULL,
  source_name TEXT NOT NULL,
  source_ref TEXT,
  discovery_id TEXT,
  company_name_raw TEXT,
  domain_raw TEXT,
  unstructured_context TEXT,
  raw_payload_json TEXT,
  normalization_status TEXT NOT NULL DEFAULT 'pending',
  dedup_key TEXT,
  ingest_order INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  FOREIGN KEY (pipeline_run_id) REFERENCES pipeline_runs(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS companies (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  canonical_name TEXT NOT NULL,
  normalized_name TEXT NOT NULL,
  canonical_domain TEXT,
  industry TEXT,
  hq_location TEXT,
  entity_confidence REAL NOT NULL DEFAULT 0,
  first_seen_run_id INTEGER,
  last_seen_run_id INTEGER,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (first_seen_run_id) REFERENCES pipeline_runs(id) ON DELETE SET NULL,
  FOREIGN KEY (last_seen_run_id) REFERENCES pipeline_runs(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS company_sources (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  company_id INTEGER NOT NULL,
  pipeline_run_id INTEGER NOT NULL,
  raw_candidate_id INTEGER,
  source_name TEXT NOT NULL,
  source_ref TEXT,
  attribution_weight REAL NOT NULL DEFAULT 1.0,
  first_seen_at TEXT NOT NULL,
  last_seen_at TEXT NOT NULL,
  FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
  FOREIGN KEY (pipeline_run_id) REFERENCES pipeline_runs(id) ON DELETE CASCADE,
  FOREIGN KEY (raw_candidate_id) REFERENCES raw_candidates(id) ON DELETE SET NULL,
  UNIQUE (company_id, pipeline_run_id, source_name, source_ref)
);

CREATE TABLE IF NOT EXISTS company_signals (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  company_id INTEGER NOT NULL,
  pipeline_run_id INTEGER NOT NULL,
  signal_type TEXT NOT NULL,
  signal_key TEXT,
  signal_value TEXT NOT NULL,
  signal_strength REAL NOT NULL DEFAULT 0,
  evidence_json TEXT,
  detected_at TEXT NOT NULL,
  FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
  FOREIGN KEY (pipeline_run_id) REFERENCES pipeline_runs(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS company_scores (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  company_id INTEGER NOT NULL,
  pipeline_run_id INTEGER NOT NULL,
  scoring_version TEXT NOT NULL,
  score INTEGER NOT NULL,
  action TEXT NOT NULL,
  confidence TEXT,
  score_reasons_json TEXT,
  created_at TEXT NOT NULL,
  FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
  FOREIGN KEY (pipeline_run_id) REFERENCES pipeline_runs(id) ON DELETE CASCADE,
  UNIQUE (company_id, pipeline_run_id, scoring_version)
);

CREATE TABLE IF NOT EXISTS company_snapshots (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  company_id INTEGER NOT NULL,
  pipeline_run_id INTEGER NOT NULL,
  scoring_version TEXT NOT NULL,
  company_name TEXT NOT NULL,
  domain TEXT,
  industry TEXT,
  hq_location TEXT,
  action TEXT NOT NULL,
  priority_score INTEGER NOT NULL,
  confidence TEXT,
  why_now TEXT,
  reason_for_fit TEXT,
  source_summary_json TEXT,
  signals_json TEXT,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
  FOREIGN KEY (pipeline_run_id) REFERENCES pipeline_runs(id) ON DELETE CASCADE,
  UNIQUE (company_id, pipeline_run_id, scoring_version)
);
