// Package store persists lead rows for the internal web UI (SQLite).
package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"salesradar/internal/domain"
	"salesradar/internal/review"
)

// Lead is a stored row (ReviewLead-style + source + timestamps).
type Lead struct {
	ID                            int64
	Company                       string
	Industry                      string
	Size                          string
	ICPMatch                      string
	DuplicateStatus               string
	LeadStatus                    string
	Confidence                    string
	Summary                       string
	Reasons                       []string
	Source                        string
	CreatedAt                     time.Time
	WebsiteDomain                 string
	LinkedInURL                   string
	CountryRegion                 string
	ReasonForFit                  string
	WhyNow                        string
	WhyNowStrength                string
	SalesAngle                    string
	PriorityScore                 int
	DataCompleteness              int
	SalesStatus                   string
	EmployeeSize                  string
	AcceptExplanation             string
	MissingOptional               []string
	SourceRef                     string
	SalesReady                    bool
	Action                        string
	OfficialDomain                string
	SourceTrace                   []string
	UsedGoogle                    bool
	UsedApollo                    bool
	UsedLinkedIn                  bool
	WebsiteEnrichmentSelectedURLs string
	WebsiteEnrichmentSummary      string
	WebsiteEnrichmentSignals      string
	WebsiteEnrichmentStatus       string
	WebsiteEnrichedAt             string
}

// ListFilter holds optional query filters and sort.
type ListFilter struct {
	Query       string
	ICPMatch    string
	LeadStatus  string
	SalesStatus string
	Industry    string
	Action      string // Contact | Research first | Ignore (empty = any)
	SortBy      string // priority, action, company, confidence, completeness
	OrderAsc    bool
}

// Open opens SQLite and applies schema.
func Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := runSQLMigrations(db, "db/migrations"); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func runSQLMigrations(db *sql.DB, dir string) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read migrations dir: %w", err)
	}
	type mig struct {
		version int
		name    string
		path    string
	}
	var list []mig
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := strings.TrimSpace(e.Name())
		if !strings.HasSuffix(strings.ToLower(name), ".sql") {
			continue
		}
		v, err := parseMigrationVersion(name)
		if err != nil {
			return err
		}
		list = append(list, mig{version: v, name: name, path: filepath.Join(dir, name)})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].version == list[j].version {
			return list[i].name < list[j].name
		}
		return list[i].version < list[j].version
	})
	for _, m := range list {
		var exists int
		if err := db.QueryRow(`SELECT 1 FROM schema_migrations WHERE version = ?`, m.version).Scan(&exists); err == nil {
			continue
		} else if err != sql.ErrNoRows {
			return fmt.Errorf("check migration %s: %w", m.name, err)
		}
		content, err := os.ReadFile(m.path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", m.name, err)
		}
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(content)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", m.name, err)
		}
		if _, err := tx.Exec(
			`INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)`,
			m.version, m.name, time.Now().UTC().Format(time.RFC3339),
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", m.name, err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func parseMigrationVersion(name string) (int, error) {
	base := strings.TrimSpace(strings.TrimSuffix(name, filepath.Ext(name)))
	if base == "" {
		return 0, fmt.Errorf("invalid migration name: %q", name)
	}
	parts := strings.SplitN(base, "_", 2)
	v, err := strconv.Atoi(parts[0])
	if err != nil || v <= 0 {
		return 0, fmt.Errorf("migration must start with positive numeric version: %s", name)
	}
	return v, nil
}

func migrate(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(leads)`)
	if err != nil {
		return err
	}
	existing := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			_ = rows.Close()
			return err
		}
		existing[name] = true
	}
	_ = rows.Close()

	alters := []struct {
		name string
		ddl  string
	}{
		{"website_domain", `ALTER TABLE leads ADD COLUMN website_domain TEXT`},
		{"linkedin_url", `ALTER TABLE leads ADD COLUMN linkedin_url TEXT`},
		{"country_region", `ALTER TABLE leads ADD COLUMN country_region TEXT`},
		{"reason_for_fit", `ALTER TABLE leads ADD COLUMN reason_for_fit TEXT`},
		{"why_now", `ALTER TABLE leads ADD COLUMN why_now TEXT`},
		{"why_now_strength", `ALTER TABLE leads ADD COLUMN why_now_strength TEXT`},
		{"sales_angle", `ALTER TABLE leads ADD COLUMN sales_angle TEXT`},
		{"priority_score", `ALTER TABLE leads ADD COLUMN priority_score INTEGER DEFAULT 0`},
		{"data_completeness", `ALTER TABLE leads ADD COLUMN data_completeness INTEGER DEFAULT 0`},
		{"sales_status", `ALTER TABLE leads ADD COLUMN sales_status TEXT`},
		{"employee_size", `ALTER TABLE leads ADD COLUMN employee_size TEXT`},
		{"accept_explanation", `ALTER TABLE leads ADD COLUMN accept_explanation TEXT`},
		{"missing_optional", `ALTER TABLE leads ADD COLUMN missing_optional TEXT`},
		{"source_ref", `ALTER TABLE leads ADD COLUMN source_ref TEXT`},
		{"sales_ready", `ALTER TABLE leads ADD COLUMN sales_ready INTEGER DEFAULT 0`},
		{"action", `ALTER TABLE leads ADD COLUMN action TEXT`},
		{"official_domain", `ALTER TABLE leads ADD COLUMN official_domain TEXT`},
		{"source_trace_json", `ALTER TABLE leads ADD COLUMN source_trace_json TEXT`},
		{"used_google", `ALTER TABLE leads ADD COLUMN used_google INTEGER DEFAULT 0`},
		{"used_apollo", `ALTER TABLE leads ADD COLUMN used_apollo INTEGER DEFAULT 0`},
		{"used_linkedin", `ALTER TABLE leads ADD COLUMN used_linkedin INTEGER DEFAULT 0`},
		{"website_enrichment_selected_urls", `ALTER TABLE leads ADD COLUMN website_enrichment_selected_urls TEXT`},
		{"website_enrichment_summary", `ALTER TABLE leads ADD COLUMN website_enrichment_summary TEXT`},
		{"website_enrichment_signals", `ALTER TABLE leads ADD COLUMN website_enrichment_signals TEXT`},
		{"website_enrichment_status", `ALTER TABLE leads ADD COLUMN website_enrichment_status TEXT`},
		{"website_enriched_at", `ALTER TABLE leads ADD COLUMN website_enriched_at TEXT`},
	}
	for _, a := range alters {
		if existing[a.name] {
			continue
		}
		if _, err := db.Exec(a.ddl); err != nil {
			return fmt.Errorf("migrate add %s: %w", a.name, err)
		}
	}
	prows, err := db.Query(`PRAGMA table_info(pipeline_runs)`)
	if err != nil {
		return err
	}
	pexisting := map[string]bool{}
	for prows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := prows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			_ = prows.Close()
			return err
		}
		pexisting[name] = true
	}
	_ = prows.Close()
	if !pexisting["run_outcome"] {
		if _, err := db.Exec(`ALTER TABLE pipeline_runs ADD COLUMN run_outcome TEXT NOT NULL DEFAULT 'success'`); err != nil {
			return fmt.Errorf("migrate add pipeline_runs.run_outcome: %w", err)
		}
	}
	return nil
}

const schema = `
CREATE TABLE IF NOT EXISTS leads (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	company TEXT,
	industry TEXT,
	size TEXT,
	icp_match TEXT,
	duplicate_status TEXT,
	lead_status TEXT,
	confidence TEXT,
	summary TEXT,
	reasons TEXT,
	source TEXT,
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_leads_icp ON leads(icp_match);
CREATE INDEX IF NOT EXISTS idx_leads_status ON leads(lead_status);
CREATE INDEX IF NOT EXISTS idx_leads_industry ON leads(industry);

CREATE TABLE IF NOT EXISTS lead_sources (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	lead_id INTEGER NOT NULL,
	source_name TEXT NOT NULL,
	source_ref TEXT,
	position INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL,
	FOREIGN KEY (lead_id) REFERENCES leads(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_lead_sources_lead_id ON lead_sources(lead_id);
CREATE INDEX IF NOT EXISTS idx_lead_sources_name ON lead_sources(source_name);

CREATE TABLE IF NOT EXISTS lead_scores (
	lead_id INTEGER PRIMARY KEY,
	icp_score INTEGER NOT NULL DEFAULT 0,
	icp_match TEXT,
	action TEXT,
	duplicate_status TEXT,
	data_completeness INTEGER NOT NULL DEFAULT 0,
	updated_at TEXT NOT NULL,
	FOREIGN KEY (lead_id) REFERENCES leads(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_lead_scores_action ON lead_scores(action);
CREATE INDEX IF NOT EXISTS idx_lead_scores_score ON lead_scores(icp_score);

CREATE TABLE IF NOT EXISTS lead_signals (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	lead_id INTEGER NOT NULL,
	signal_type TEXT NOT NULL,
	signal_value TEXT NOT NULL,
	created_at TEXT NOT NULL,
	FOREIGN KEY (lead_id) REFERENCES leads(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_lead_signals_lead_id ON lead_signals(lead_id);
CREATE INDEX IF NOT EXISTS idx_lead_signals_type ON lead_signals(signal_type);

CREATE TABLE IF NOT EXISTS pipeline_runs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	run_uuid TEXT NOT NULL UNIQUE,
	triggered_by TEXT,
	discovery_mode TEXT NOT NULL,
	scoring_version TEXT NOT NULL,
	started_at TEXT NOT NULL,
	finished_at TEXT,
	status TEXT NOT NULL DEFAULT 'running',
	run_outcome TEXT NOT NULL DEFAULT 'success',
	total_raw_candidates INTEGER NOT NULL DEFAULT 0,
	total_companies INTEGER NOT NULL DEFAULT 0,
	total_snapshots INTEGER NOT NULL DEFAULT 0,
	error_message TEXT,
	run_debug_json TEXT
);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_started_at ON pipeline_runs(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_status ON pipeline_runs(status);

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
CREATE INDEX IF NOT EXISTS idx_raw_candidates_run_id ON raw_candidates(pipeline_run_id);
CREATE INDEX IF NOT EXISTS idx_raw_candidates_source ON raw_candidates(source_name);
CREATE INDEX IF NOT EXISTS idx_raw_candidates_domain ON raw_candidates(domain_raw);

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
CREATE UNIQUE INDEX IF NOT EXISTS idx_companies_domain_unique ON companies(canonical_domain) WHERE canonical_domain IS NOT NULL AND canonical_domain != '';
CREATE INDEX IF NOT EXISTS idx_companies_name ON companies(normalized_name);
CREATE INDEX IF NOT EXISTS idx_companies_industry ON companies(industry);

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
CREATE INDEX IF NOT EXISTS idx_company_sources_company_run ON company_sources(company_id, pipeline_run_id);
CREATE INDEX IF NOT EXISTS idx_company_sources_source ON company_sources(source_name);

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
CREATE INDEX IF NOT EXISTS idx_company_signals_company_run ON company_signals(company_id, pipeline_run_id);
CREATE INDEX IF NOT EXISTS idx_company_signals_type ON company_signals(signal_type);

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
CREATE INDEX IF NOT EXISTS idx_company_scores_company_run ON company_scores(company_id, pipeline_run_id);
CREATE INDEX IF NOT EXISTS idx_company_scores_action ON company_scores(action);
CREATE INDEX IF NOT EXISTS idx_company_scores_score ON company_scores(score DESC);

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
CREATE INDEX IF NOT EXISTS idx_company_snapshots_pipeline_run_id ON company_snapshots(pipeline_run_id);
CREATE INDEX IF NOT EXISTS idx_company_snapshots_action ON company_snapshots(action);
CREATE INDEX IF NOT EXISTS idx_company_snapshots_priority_score ON company_snapshots(priority_score DESC);
CREATE INDEX IF NOT EXISTS idx_company_snapshots_industry ON company_snapshots(industry);

CREATE TABLE IF NOT EXISTS discovery_source_prefs (
	source_key TEXT PRIMARY KEY,
	enabled INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS app_kv (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
`

// PipelineRunRecord is one row from pipeline_runs (used for debug / latest run view).
type PipelineRunRecord struct {
	ID            int64
	RunUUID       string
	StartedAt     string
	FinishedAt    sql.NullString
	Status        string
	RunOutcome    string
	DiscoveryMode string
	RunDebugJSON  sql.NullString
	ErrorMessage  sql.NullString
}

// LatestPipelineRun returns the most recently started pipeline run, if any.
func LatestPipelineRun(db *sql.DB) (PipelineRunRecord, error) {
	var r PipelineRunRecord
	err := db.QueryRow(`
		SELECT id, run_uuid, started_at, finished_at, status, COALESCE(run_outcome,''), discovery_mode, run_debug_json, error_message
		FROM pipeline_runs
		ORDER BY started_at DESC, id DESC
		LIMIT 1
	`).Scan(&r.ID, &r.RunUUID, &r.StartedAt, &r.FinishedAt, &r.Status, &r.RunOutcome, &r.DiscoveryMode, &r.RunDebugJSON, &r.ErrorMessage)
	if err != nil {
		return PipelineRunRecord{}, err
	}
	return r, nil
}

// ReplaceAll deletes existing rows and inserts new leads (single run snapshot).
func ReplaceAll(db *sql.DB, rows []LeadInput, runDebugJSON string, runOutcome string) (stored int, err error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if _, err = tx.Exec(`DELETE FROM leads`); err != nil {
		return 0, err
	}
	if _, err = tx.Exec(`DELETE FROM lead_sources`); err != nil {
		return 0, err
	}
	if _, err = tx.Exec(`DELETE FROM lead_scores`); err != nil {
		return 0, err
	}
	if _, err = tx.Exec(`DELETE FROM lead_signals`); err != nil {
		return 0, err
	}
	runID, err := createPipelineRun(tx, rows)
	if err != nil {
		return 0, err
	}
	companyByKey := map[string]int64{}
	for _, r := range rows {
		reasonsJSON, err := json.Marshal(r.Reasons)
		if err != nil {
			return 0, err
		}
		missJSON, err := json.Marshal(r.MissingOptional)
		if err != nil {
			return 0, err
		}
		sr := 0
		if r.SalesReady {
			sr = 1
		}
		ug, ua, ul := 0, 0, 0
		if r.UsedGoogle {
			ug = 1
		}
		if r.UsedApollo {
			ua = 1
		}
		if r.UsedLinkedIn {
			ul = 1
		}
		traceJSON, err := json.Marshal(r.SourceTrace)
		if err != nil {
			return 0, err
		}
		res, err := tx.Exec(`
			INSERT INTO leads (company, industry, size, icp_match, duplicate_status, lead_status, confidence, summary, reasons, source, created_at,
				website_domain, linkedin_url, country_region, reason_for_fit, why_now, why_now_strength, sales_angle, priority_score, data_completeness, sales_status, employee_size, accept_explanation, missing_optional, source_ref, sales_ready, action,
				official_domain, source_trace_json, used_google, used_apollo, used_linkedin,
				website_enrichment_selected_urls, website_enrichment_summary, website_enrichment_signals, website_enrichment_status, website_enriched_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			r.Company, r.Industry, r.Size, r.ICPMatch, r.DuplicateStatus, r.LeadStatus, r.Confidence, r.Summary, string(reasonsJSON), r.Source, r.CreatedAt.UTC().Format(time.RFC3339),
			r.WebsiteDomain, r.LinkedInURL, r.CountryRegion, r.ReasonForFit, r.WhyNow, r.WhyNowStrength, r.SalesAngle, r.PriorityScore, r.DataCompleteness, r.SalesStatus, r.EmployeeSize, r.AcceptExplanation, string(missJSON), r.SourceRef, sr, r.Action,
			r.OfficialDomain, string(traceJSON), ug, ua, ul,
			r.WebsiteEnrichmentSelectedURLs, r.WebsiteEnrichmentSummary, r.WebsiteEnrichmentSignals, r.WebsiteEnrichmentStatus, r.WebsiteEnrichedAt,
		)
		if err != nil {
			return 0, err
		}
		leadID, err := res.LastInsertId()
		if err != nil {
			return 0, err
		}
		if err := insertLeadSources(tx, leadID, r); err != nil {
			return 0, err
		}
		if err := insertLeadScores(tx, leadID, r); err != nil {
			return 0, err
		}
		if err := insertLeadSignals(tx, leadID, r); err != nil {
			return 0, err
		}
		rawID, err := insertRawCandidate(tx, runID, r, stored)
		if err != nil {
			return 0, err
		}
		companyID, err := upsertCompany(tx, runID, companyByKey, r)
		if err != nil {
			return 0, err
		}
		if err := insertCompanySource(tx, companyID, runID, rawID, r); err != nil {
			return 0, err
		}
		if err := insertCompanySignals(tx, companyID, runID, r); err != nil {
			return 0, err
		}
		if err := insertCompanyScore(tx, companyID, runID, r); err != nil {
			return 0, err
		}
		if err := upsertCompanySnapshot(tx, companyID, runID, r); err != nil {
			return 0, err
		}
		stored++
	}
	if err := finalizePipelineRun(tx, runID, stored, len(companyByKey), runDebugJSON, runOutcome); err != nil {
		return 0, err
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return stored, nil
}

func createPipelineRun(tx *sql.Tx, rows []LeadInput) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	runUUID := fmt.Sprintf("run-%d", time.Now().UTC().UnixNano())
	mode := "multi_source"
	if _, err := tx.Exec(
		`INSERT INTO pipeline_runs (run_uuid, triggered_by, discovery_mode, scoring_version, started_at, status, run_outcome, total_raw_candidates)
		 VALUES (?, ?, ?, ?, ?, 'running', 'running', ?)`,
		runUUID, "web_run", mode, currentScoringVersion(), now, len(rows),
	); err != nil {
		return 0, err
	}
	var id int64
	if err := tx.QueryRow(`SELECT id FROM pipeline_runs WHERE run_uuid = ?`, runUUID).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func finalizePipelineRun(tx *sql.Tx, runID int64, snapshots int, companies int, runDebugJSON string, runOutcome string) error {
	var dbg any
	if strings.TrimSpace(runDebugJSON) == "" {
		dbg = nil
	} else {
		dbg = runDebugJSON
	}
	_, err := tx.Exec(
		`UPDATE pipeline_runs
		 SET finished_at = ?, status = 'succeeded', run_outcome = ?, total_companies = ?, total_snapshots = ?, run_debug_json = ?
		 WHERE id = ?`,
		time.Now().UTC().Format(time.RFC3339), strings.TrimSpace(runOutcome), companies, snapshots, dbg, runID,
	)
	return err
}

func currentScoringVersion() string {
	return "icp-v1"
}

// RecordFailedPipelineRun persists a failed run row when the pipeline cannot produce output rows.
func RecordFailedPipelineRun(db *sql.DB, discoveryMode, errMsg, runOutcome string) error {
	runUUID := fmt.Sprintf("run-%d", time.Now().UTC().UnixNano())
	now := time.Now().UTC().Format(time.RFC3339)
	if strings.TrimSpace(discoveryMode) == "" {
		discoveryMode = "multi_source"
	}
	_, err := db.Exec(
		`INSERT INTO pipeline_runs (run_uuid, triggered_by, discovery_mode, scoring_version, started_at, finished_at, status, run_outcome, error_message, total_raw_candidates, total_companies, total_snapshots)
		 VALUES (?, ?, ?, ?, ?, ?, 'failed', ?, ?, 0, 0, 0)`,
		runUUID, "web_run", strings.TrimSpace(discoveryMode), currentScoringVersion(), now, now, strings.TrimSpace(runOutcome), strings.TrimSpace(errMsg),
	)
	return err
}

func insertRawCandidate(tx *sql.Tx, runID int64, in LeadInput, order int) (int64, error) {
	rawPayload, _ := json.Marshal(map[string]any{
		"source":       in.Source,
		"source_ref":   in.SourceRef,
		"source_trace": in.SourceTrace,
		"reasons":      in.Reasons,
	})
	name := strings.TrimSpace(in.Company)
	sourceName := domain.PrimaryDiscoverySourceNameFromTrace(in.SourceTrace, domain.Source(in.Source))
	res, err := tx.Exec(
		`INSERT INTO raw_candidates (pipeline_run_id, source_name, source_ref, discovery_id, company_name_raw, domain_raw, unstructured_context, raw_payload_json, normalization_status, dedup_key, ingest_order, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'normalized', ?, ?, ?)`,
		runID, sourceName, in.SourceRef, "", name, strings.TrimSpace(in.OfficialDomain), in.Summary, string(rawPayload), dedupCompanyKey(name, in.OfficialDomain), order, time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func upsertCompany(tx *sql.Tx, runID int64, cache map[string]int64, in LeadInput) (int64, error) {
	name := strings.TrimSpace(in.Company)
	if name == "" {
		name = "Unknown Company"
	}
	domain := strings.TrimSpace(strings.ToLower(in.OfficialDomain))
	key := dedupCompanyKey(name, domain)
	if id, ok := cache[key]; ok {
		return id, nil
	}
	var existingID int64
	if domain != "" {
		if err := tx.QueryRow(`SELECT id FROM companies WHERE canonical_domain = ?`, domain).Scan(&existingID); err == nil {
			cache[key] = existingID
			_, _ = tx.Exec(`UPDATE companies SET canonical_name=?, normalized_name=?, industry=?, hq_location=?, last_seen_run_id=?, updated_at=? WHERE id=?`,
				name, normalizeCompanyName(name), emptyToNullText(in.Industry), emptyToNullText(in.CountryRegion), runID, time.Now().UTC().Format(time.RFC3339), existingID)
			return existingID, nil
		} else if err != sql.ErrNoRows {
			return 0, err
		}
	}
	res, err := tx.Exec(
		`INSERT INTO companies (canonical_name, normalized_name, canonical_domain, industry, hq_location, entity_confidence, first_seen_run_id, last_seen_run_id, created_at, updated_at)
		 VALUES (?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?, ?, ?, ?, ?)`,
		name, normalizeCompanyName(name), domain, strings.TrimSpace(in.Industry), strings.TrimSpace(in.CountryRegion), confidenceToFloat(in.Confidence), runID, runID, time.Now().UTC().Format(time.RFC3339), time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	cache[key] = id
	return id, nil
}

func insertCompanySource(tx *sql.Tx, companyID, runID, rawID int64, in LeadInput) error {
	now := time.Now().UTC().Format(time.RFC3339)
	insertOne := func(sourceName, sourceRef string) error {
		sourceName = strings.TrimSpace(sourceName)
		if sourceName == "" {
			return nil
		}
		_, err := tx.Exec(
			`INSERT OR IGNORE INTO company_sources (company_id, pipeline_run_id, raw_candidate_id, source_name, source_ref, attribution_weight, first_seen_at, last_seen_at)
			 VALUES (?, ?, ?, ?, ?, 1.0, ?, ?)`,
			companyID, runID, rawID, sourceName, sourceRef, now, now,
		)
		return err
	}
	for _, s := range in.SourceTrace {
		if err := insertOne(s, in.SourceRef); err != nil {
			return err
		}
	}
	return insertOne(in.Source, in.SourceRef)
}

func insertCompanySignals(tx *sql.Tx, companyID, runID int64, in LeadInput) error {
	now := time.Now().UTC().Format(time.RFC3339)
	ins := func(t, k, v string, strength float64) error {
		v = strings.TrimSpace(v)
		if v == "" {
			return nil
		}
		_, err := tx.Exec(
			`INSERT INTO company_signals (company_id, pipeline_run_id, signal_type, signal_key, signal_value, signal_strength, evidence_json, detected_at)
			 VALUES (?, ?, ?, NULLIF(?, ''), ?, ?, NULL, ?)`,
			companyID, runID, t, k, v, strength, now,
		)
		return err
	}
	if err := ins("why_now", "", in.WhyNow, whyNowStrengthToFloat(in.WhyNowStrength)); err != nil {
		return err
	}
	for _, r := range in.Reasons {
		if err := ins("icp_reason", "", r, 0.5); err != nil {
			return err
		}
	}
	return nil
}

func insertCompanyScore(tx *sql.Tx, companyID, runID int64, in LeadInput) error {
	reasonsJSON, _ := json.Marshal(in.Reasons)
	_, err := tx.Exec(
		`INSERT OR REPLACE INTO company_scores (company_id, pipeline_run_id, scoring_version, score, action, confidence, score_reasons_json, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		companyID, runID, currentScoringVersion(), in.PriorityScore, in.Action, in.Confidence, string(reasonsJSON), time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

func upsertCompanySnapshot(tx *sql.Tx, companyID, runID int64, in LeadInput) error {
	sourceJSON, _ := json.Marshal(in.SourceTrace)
	signalsJSON, _ := json.Marshal(in.Reasons)
	_, err := tx.Exec(
		`INSERT OR REPLACE INTO company_snapshots (company_id, pipeline_run_id, scoring_version, company_name, domain, industry, hq_location, action, priority_score, confidence, why_now, reason_for_fit, source_summary_json, signals_json, updated_at)
		 VALUES (?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, ?, ?)`,
		companyID, runID, currentScoringVersion(), strings.TrimSpace(in.Company), strings.TrimSpace(in.OfficialDomain), strings.TrimSpace(in.Industry), strings.TrimSpace(in.CountryRegion),
		in.Action, in.PriorityScore, in.Confidence, in.WhyNow, in.ReasonForFit, string(sourceJSON), string(signalsJSON), time.Now().UTC().Format(time.RFC3339),
	)
	return err
}

func dedupCompanyKey(name, domain string) string {
	domain = strings.TrimSpace(strings.ToLower(domain))
	if domain != "" {
		return "d:" + domain
	}
	return "n:" + normalizeCompanyName(name)
}

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

func normalizeCompanyName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = nonAlphaNum.ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(s), " ")
}

func confidenceToFloat(s string) float64 {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "high":
		return 0.9
	case "medium":
		return 0.6
	default:
		return 0.3
	}
}

func whyNowStrengthToFloat(s string) float64 {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "high":
		return 0.9
	case "medium":
		return 0.6
	default:
		return 0.3
	}
}

func emptyToNullText(s string) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return s
}

func insertLeadSources(tx *sql.Tx, leadID int64, in LeadInput) error {
	seen := map[string]struct{}{}
	now := in.CreatedAt.UTC().Format(time.RFC3339)
	pos := 0
	for _, src := range in.SourceTrace {
		src = strings.TrimSpace(src)
		if src == "" {
			continue
		}
		if _, ok := seen[src]; ok {
			continue
		}
		seen[src] = struct{}{}
		if _, err := tx.Exec(
			`INSERT INTO lead_sources (lead_id, source_name, source_ref, position, created_at) VALUES (?, ?, ?, ?, ?)`,
			leadID, src, in.SourceRef, pos, now,
		); err != nil {
			return err
		}
		pos++
	}
	if pos == 0 {
		src := strings.TrimSpace(in.Source)
		if src == "" {
			src = "unknown_source"
		}
		if _, err := tx.Exec(
			`INSERT INTO lead_sources (lead_id, source_name, source_ref, position, created_at) VALUES (?, ?, ?, 0, ?)`,
			leadID, src, in.SourceRef, now,
		); err != nil {
			return err
		}
	}
	return nil
}

func insertLeadScores(tx *sql.Tx, leadID int64, in LeadInput) error {
	_, err := tx.Exec(
		`INSERT INTO lead_scores (lead_id, icp_score, icp_match, action, duplicate_status, data_completeness, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		leadID, in.PriorityScore, in.ICPMatch, in.Action, in.DuplicateStatus, in.DataCompleteness, in.CreatedAt.UTC().Format(time.RFC3339),
	)
	return err
}

func insertLeadSignals(tx *sql.Tx, leadID int64, in LeadInput) error {
	now := in.CreatedAt.UTC().Format(time.RFC3339)
	ins := func(t, v string) error {
		v = strings.TrimSpace(v)
		if v == "" {
			return nil
		}
		_, err := tx.Exec(
			`INSERT INTO lead_signals (lead_id, signal_type, signal_value, created_at) VALUES (?, ?, ?, ?)`,
			leadID, t, v, now,
		)
		return err
	}
	for _, r := range in.Reasons {
		if err := ins("icp_reason", r); err != nil {
			return err
		}
	}
	if err := ins("why_now", in.WhyNow); err != nil {
		return err
	}
	if err := ins("why_now_strength", in.WhyNowStrength); err != nil {
		return err
	}
	if in.UsedGoogle {
		if err := ins("integration", "google"); err != nil {
			return err
		}
	}
	if in.UsedApollo {
		if err := ins("integration", "apollo"); err != nil {
			return err
		}
	}
	if in.UsedLinkedIn {
		if err := ins("integration", "linkedin"); err != nil {
			return err
		}
	}
	return nil
}

// LeadInput is one row to persist (from ReviewLead + source).
type LeadInput struct {
	Company                       string
	Industry                      string
	Size                          string
	ICPMatch                      string
	DuplicateStatus               string
	LeadStatus                    string
	Confidence                    string
	Summary                       string
	Reasons                       []string
	Source                        string
	CreatedAt                     time.Time
	WebsiteDomain                 string
	LinkedInURL                   string
	CountryRegion                 string
	ReasonForFit                  string
	WhyNow                        string
	WhyNowStrength                string
	SalesAngle                    string
	PriorityScore                 int
	DataCompleteness              int
	SalesStatus                   string
	EmployeeSize                  string
	AcceptExplanation             string
	MissingOptional               []string
	SourceRef                     string
	SalesReady                    bool
	Action                        string
	OfficialDomain                string
	SourceTrace                   []string
	UsedGoogle                    bool
	UsedApollo                    bool
	UsedLinkedIn                  bool
	WebsiteEnrichmentSelectedURLs string
	WebsiteEnrichmentSummary      string
	WebsiteEnrichmentSignals      string
	WebsiteEnrichmentStatus       string
	WebsiteEnrichedAt             string
}

// FromStaged builds storage input from pipeline output.
func FromStaged(staged domain.StagedOdooLead, r review.ReviewLead) LeadInput {
	company, industry := "", ""
	if r.Company != nil {
		company = *r.Company
	}
	if r.Industry != nil {
		industry = *r.Industry
	}
	li := LeadInput{
		Company:           company,
		Industry:          industry,
		Size:              r.Size,
		ICPMatch:          r.ICPMatch,
		DuplicateStatus:   r.DuplicateStatus,
		LeadStatus:        r.LeadStatus,
		Confidence:        r.Confidence,
		Summary:           r.Summary,
		Reasons:           append([]string(nil), r.Reasons...),
		Source:            string(staged.Source),
		CreatedAt:         time.Now(),
		WebsiteDomain:     r.WebsiteDomain,
		LinkedInURL:       r.LinkedInURL,
		CountryRegion:     r.CountryRegion,
		ReasonForFit:      r.ReasonForFit,
		WhyNow:            r.WhyNow,
		WhyNowStrength:    r.WhyNowStrength,
		SalesAngle:        r.SalesAngle,
		PriorityScore:     r.PriorityScore,
		DataCompleteness:  r.DataCompleteness,
		SalesStatus:       r.SalesStatus,
		EmployeeSize:      r.EmployeeSize,
		AcceptExplanation: r.AcceptExplanation,
		MissingOptional:   append([]string(nil), r.MissingOptional...),
		SourceRef:         staged.SourceRef,
		SalesReady:        r.SalesReady,
		Action:            r.Action,
		OfficialDomain:    r.OfficialDomain,
		SourceTrace:       append([]string(nil), r.SourceTrace...),
		UsedGoogle:        r.UsedGoogle,
		UsedApollo:        r.UsedApollo,
		UsedLinkedIn:      r.UsedLinkedIn,
	}
	if we := staged.WebsiteEnrichment; we != nil {
		if b, err := json.Marshal(we.SelectedURLs); err == nil {
			li.WebsiteEnrichmentSelectedURLs = string(b)
		}
		li.WebsiteEnrichmentSummary = we.Summary
		li.WebsiteEnrichmentSignals = we.Signals
		li.WebsiteEnrichmentStatus = we.Status
		li.WebsiteEnrichedAt = we.EnrichedAt
	}
	return li
}

const leadSelect = `id, company, industry, size, icp_match, duplicate_status, lead_status, confidence, summary, reasons, source, created_at,
		COALESCE(website_domain,''), COALESCE(linkedin_url,''), COALESCE(country_region,''), COALESCE(reason_for_fit,''), COALESCE(why_now,''), COALESCE(why_now_strength,''), COALESCE(sales_angle,''), COALESCE(priority_score,0),
		COALESCE(data_completeness,0), COALESCE(sales_status,''), COALESCE(employee_size,''), COALESCE(accept_explanation,''), COALESCE(missing_optional,''), COALESCE(source_ref,''),
		COALESCE(sales_ready,0), COALESCE(action,''),
		COALESCE(official_domain,''), COALESCE(source_trace_json,''), COALESCE(used_google,0), COALESCE(used_apollo,0), COALESCE(used_linkedin,0),
		COALESCE(website_enrichment_selected_urls,''), COALESCE(website_enrichment_summary,''), COALESCE(website_enrichment_signals,''), COALESCE(website_enrichment_status,''), COALESCE(website_enriched_at,'')`

// List returns leads matching filters.
func List(db *sql.DB, f ListFilter) ([]Lead, error) {
	useSnapshots, err := snapshotReadEnabled(db)
	if err != nil {
		return nil, err
	}
	if useSnapshots {
		return listFromSnapshots(db, f)
	}
	return listFromLeads(db, f)
}

func listFromLeads(db *sql.DB, f ListFilter) ([]Lead, error) {
	var args []any
	var where []string
	if q := strings.TrimSpace(f.Query); q != "" {
		pat := "%" + strings.ToLower(q) + "%"
		where = append(where, `(lower(company) LIKE ? OR lower(industry) LIKE ? OR lower(summary) LIKE ? OR lower(COALESCE(reason_for_fit,'')) LIKE ? OR lower(COALESCE(country_region,'')) LIKE ? OR lower(COALESCE(website_domain,'')) LIKE ? OR lower(COALESCE(official_domain,'')) LIKE ? OR lower(COALESCE(source_trace_json,'')) LIKE ?)`)
		args = append(args, pat, pat, pat, pat, pat, pat, pat, pat)
	}
	if f.ICPMatch != "" {
		where = append(where, `icp_match = ?`)
		args = append(args, f.ICPMatch)
	}
	if f.LeadStatus != "" {
		where = append(where, `lead_status = ?`)
		args = append(args, f.LeadStatus)
	}
	if f.SalesStatus != "" {
		where = append(where, `sales_status = ?`)
		args = append(args, f.SalesStatus)
	}
	if f.Industry != "" {
		where = append(where, `lower(industry) = lower(?)`)
		args = append(args, f.Industry)
	}
	if f.Action != "" {
		where = append(where, `action = ?`)
		args = append(args, f.Action)
	}
	q := `SELECT ` + leadSelect + ` FROM leads`
	if len(where) > 0 {
		q += ` WHERE ` + strings.Join(where, ` AND `)
	}
	order := `DESC`
	if f.OrderAsc {
		order = `ASC`
	}
	switch f.SortBy {
	case "priority":
		q += ` ORDER BY COALESCE(priority_score,0) DESC, data_completeness DESC, lower(company) ASC`
	case "confidence":
		q += fmt.Sprintf(` ORDER BY CASE confidence WHEN 'high' THEN 3 WHEN 'medium' THEN 2 WHEN 'low' THEN 1 ELSE 0 END %s, company ASC`, order)
	case "completeness":
		q += fmt.Sprintf(` ORDER BY data_completeness %s, lower(company) ASC`, order)
	case "action":
		if f.OrderAsc {
			q += ` ORDER BY CASE COALESCE(action,'') WHEN 'Contact' THEN 1 WHEN 'Research first' THEN 2 WHEN 'Ignore' THEN 3 ELSE 4 END ASC, lower(company) ASC, id ASC`
		} else {
			q += ` ORDER BY CASE COALESCE(action,'') WHEN 'Contact' THEN 3 WHEN 'Research first' THEN 2 WHEN 'Ignore' THEN 1 ELSE 4 END ASC, lower(company) ASC, id ASC`
		}
	default:
		q += fmt.Sprintf(` ORDER BY lower(company) %s, id ASC`, order)
	}
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Lead
	for rows.Next() {
		l, err := scanLead(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func listFromSnapshots(db *sql.DB, f ListFilter) ([]Lead, error) {
	var latestRunID int64
	if err := db.QueryRow(`SELECT id FROM pipeline_runs ORDER BY started_at DESC, id DESC LIMIT 1`).Scan(&latestRunID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	var args []any
	var where []string
	where = append(where, `pipeline_run_id = ?`)
	args = append(args, latestRunID)
	if q := strings.TrimSpace(f.Query); q != "" {
		pat := "%" + strings.ToLower(q) + "%"
		where = append(where, `(lower(company_name) LIKE ? OR lower(COALESCE(industry,'')) LIKE ? OR lower(COALESCE(reason_for_fit,'')) LIKE ? OR lower(COALESCE(hq_location,'')) LIKE ? OR lower(COALESCE(domain,'')) LIKE ? OR lower(COALESCE(source_summary_json,'')) LIKE ?)`)
		args = append(args, pat, pat, pat, pat, pat, pat)
	}
	if f.Action != "" {
		where = append(where, `action = ?`)
		args = append(args, f.Action)
	}
	if f.Industry != "" {
		where = append(where, `lower(industry) = lower(?)`)
		args = append(args, f.Industry)
	}
	if f.ICPMatch != "" {
		switch f.ICPMatch {
		case "high":
			where = append(where, `priority_score >= 80`)
		case "medium":
			where = append(where, `priority_score >= 50 AND priority_score < 80`)
		case "low":
			where = append(where, `priority_score < 50`)
		}
	}
	q := `SELECT id, company_name, COALESCE(industry,''), COALESCE(domain,''), COALESCE(hq_location,''), COALESCE(action,''), COALESCE(priority_score,0), COALESCE(confidence,''), COALESCE(why_now,''), COALESCE(reason_for_fit,''), COALESCE(source_summary_json,''), COALESCE(signals_json,''), updated_at FROM company_snapshots`
	if len(where) > 0 {
		q += ` WHERE ` + strings.Join(where, ` AND `)
	}
	switch f.SortBy {
	case "priority":
		q += ` ORDER BY priority_score DESC, lower(company_name) ASC`
	case "action":
		if f.OrderAsc {
			q += ` ORDER BY CASE COALESCE(action,'') WHEN 'Contact' THEN 1 WHEN 'Research first' THEN 2 WHEN 'Ignore' THEN 3 ELSE 4 END ASC, lower(company_name) ASC, id ASC`
		} else {
			q += ` ORDER BY CASE COALESCE(action,'') WHEN 'Contact' THEN 3 WHEN 'Research first' THEN 2 WHEN 'Ignore' THEN 1 ELSE 4 END ASC, lower(company_name) ASC, id ASC`
		}
	case "confidence":
		order := "DESC"
		if f.OrderAsc {
			order = "ASC"
		}
		q += fmt.Sprintf(` ORDER BY CASE confidence WHEN 'high' THEN 3 WHEN 'medium' THEN 2 WHEN 'low' THEN 1 ELSE 0 END %s, lower(company_name) ASC`, order)
	case "completeness":
		// Not directly available in snapshots; keep deterministic order.
		q += ` ORDER BY priority_score DESC, lower(company_name) ASC`
	default:
		order := "DESC"
		if f.OrderAsc {
			order = "ASC"
		}
		q += fmt.Sprintf(` ORDER BY lower(company_name) %s, id ASC`, order)
	}
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Lead
	for rows.Next() {
		var l Lead
		var sourceRaw, signalsRaw, updatedRaw string
		if err := rows.Scan(&l.ID, &l.Company, &l.Industry, &l.OfficialDomain, &l.CountryRegion, &l.Action, &l.PriorityScore, &l.Confidence, &l.WhyNow, &l.ReasonForFit, &sourceRaw, &signalsRaw, &updatedRaw); err != nil {
			return nil, err
		}
		l.WebsiteDomain = l.OfficialDomain
		l.SalesStatus = salesStatusFromAction(l.Action)
		l.LeadStatus = leadStatusFromAction(l.Action)
		l.ICPMatch = icpMatchFromScore(l.PriorityScore)
		l.Summary = l.ReasonForFit
		l.SalesReady = l.Action == "Contact"
		_ = json.Unmarshal([]byte(sourceRaw), &l.SourceTrace)
		_ = json.Unmarshal([]byte(signalsRaw), &l.Reasons)
		l.CreatedAt, _ = time.Parse(time.RFC3339, updatedRaw)
		out = append(out, l)
	}
	return out, rows.Err()
}

func scanLead(rows *sql.Rows) (Lead, error) {
	var l Lead
	var reasonsRaw, missRaw string
	var created string
	var sr, ug, ua, ul int
	var traceRaw string
	err := rows.Scan(
		&l.ID, &l.Company, &l.Industry, &l.Size, &l.ICPMatch, &l.DuplicateStatus, &l.LeadStatus, &l.Confidence, &l.Summary, &reasonsRaw, &l.Source, &created,
		&l.WebsiteDomain, &l.LinkedInURL, &l.CountryRegion, &l.ReasonForFit, &l.WhyNow, &l.WhyNowStrength, &l.SalesAngle, &l.PriorityScore, &l.DataCompleteness, &l.SalesStatus, &l.EmployeeSize, &l.AcceptExplanation, &missRaw, &l.SourceRef,
		&sr, &l.Action,
		&l.OfficialDomain, &traceRaw, &ug, &ua, &ul,
		&l.WebsiteEnrichmentSelectedURLs, &l.WebsiteEnrichmentSummary, &l.WebsiteEnrichmentSignals, &l.WebsiteEnrichmentStatus, &l.WebsiteEnrichedAt,
	)
	if err != nil {
		return l, err
	}
	l.SalesReady = sr != 0
	l.UsedGoogle = ug != 0
	l.UsedApollo = ua != 0
	l.UsedLinkedIn = ul != 0
	_ = json.Unmarshal([]byte(reasonsRaw), &l.Reasons)
	if strings.TrimSpace(traceRaw) != "" {
		_ = json.Unmarshal([]byte(traceRaw), &l.SourceTrace)
	}
	if missRaw != "" {
		_ = json.Unmarshal([]byte(missRaw), &l.MissingOptional)
	}
	l.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return l, nil
}

// Get returns one lead by id.
func Get(db *sql.DB, id int64) (Lead, error) {
	useSnapshots, err := snapshotReadEnabled(db)
	if err != nil {
		return Lead{}, err
	}
	if useSnapshots {
		if l, err := getFromSnapshots(db, id); err == nil {
			return l, nil
		} else if err != sql.ErrNoRows {
			return l, err
		}
	}
	return getFromLeads(db, id)
}

func getFromLeads(db *sql.DB, id int64) (Lead, error) {
	row := db.QueryRow(`SELECT `+leadSelect+` FROM leads WHERE id = ?`, id)
	var l Lead
	var reasonsRaw, missRaw string
	var created string
	var sr, ug, ua, ul int
	var traceRaw string
	err := row.Scan(
		&l.ID, &l.Company, &l.Industry, &l.Size, &l.ICPMatch, &l.DuplicateStatus, &l.LeadStatus, &l.Confidence, &l.Summary, &reasonsRaw, &l.Source, &created,
		&l.WebsiteDomain, &l.LinkedInURL, &l.CountryRegion, &l.ReasonForFit, &l.WhyNow, &l.WhyNowStrength, &l.SalesAngle, &l.PriorityScore, &l.DataCompleteness, &l.SalesStatus, &l.EmployeeSize, &l.AcceptExplanation, &missRaw, &l.SourceRef,
		&sr, &l.Action,
		&l.OfficialDomain, &traceRaw, &ug, &ua, &ul,
		&l.WebsiteEnrichmentSelectedURLs, &l.WebsiteEnrichmentSummary, &l.WebsiteEnrichmentSignals, &l.WebsiteEnrichmentStatus, &l.WebsiteEnrichedAt,
	)
	if err != nil {
		return l, err
	}
	l.SalesReady = sr != 0
	l.UsedGoogle = ug != 0
	l.UsedApollo = ua != 0
	l.UsedLinkedIn = ul != 0
	_ = json.Unmarshal([]byte(reasonsRaw), &l.Reasons)
	if strings.TrimSpace(traceRaw) != "" {
		_ = json.Unmarshal([]byte(traceRaw), &l.SourceTrace)
	}
	if missRaw != "" {
		_ = json.Unmarshal([]byte(missRaw), &l.MissingOptional)
	}
	l.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return l, nil
}

func getFromSnapshots(db *sql.DB, id int64) (Lead, error) {
	row := db.QueryRow(`SELECT id, company_name, COALESCE(industry,''), COALESCE(domain,''), COALESCE(hq_location,''), COALESCE(action,''), COALESCE(priority_score,0), COALESCE(confidence,''), COALESCE(why_now,''), COALESCE(reason_for_fit,''), COALESCE(source_summary_json,''), COALESCE(signals_json,''), updated_at FROM company_snapshots WHERE id = ?`, id)
	var l Lead
	var sourceRaw, signalsRaw, updatedRaw string
	if err := row.Scan(&l.ID, &l.Company, &l.Industry, &l.OfficialDomain, &l.CountryRegion, &l.Action, &l.PriorityScore, &l.Confidence, &l.WhyNow, &l.ReasonForFit, &sourceRaw, &signalsRaw, &updatedRaw); err != nil {
		return l, err
	}
	l.WebsiteDomain = l.OfficialDomain
	l.SalesStatus = salesStatusFromAction(l.Action)
	l.LeadStatus = leadStatusFromAction(l.Action)
	l.ICPMatch = icpMatchFromScore(l.PriorityScore)
	l.Summary = l.ReasonForFit
	l.SalesReady = l.Action == "Contact"
	_ = json.Unmarshal([]byte(sourceRaw), &l.SourceTrace)
	_ = json.Unmarshal([]byte(signalsRaw), &l.Reasons)
	l.CreatedAt, _ = time.Parse(time.RFC3339, updatedRaw)
	return l, nil
}

// Count returns total rows.
func Count(db *sql.DB) (int, error) {
	useSnapshots, err := snapshotReadEnabled(db)
	if err != nil {
		return 0, err
	}
	if useSnapshots {
		var n int
		err := db.QueryRow(`SELECT COUNT(*) FROM company_snapshots WHERE pipeline_run_id = (SELECT id FROM pipeline_runs ORDER BY started_at DESC, id DESC LIMIT 1)`).Scan(&n)
		return n, err
	}
	var n int
	err = db.QueryRow(`SELECT COUNT(*) FROM leads`).Scan(&n)
	return n, err
}

// DistinctIndustries returns non-empty industry values for filter dropdowns.
func DistinctIndustries(db *sql.DB) ([]string, error) {
	useSnapshots, err := snapshotReadEnabled(db)
	if err != nil {
		return nil, err
	}
	q := `SELECT DISTINCT industry FROM leads WHERE industry != '' ORDER BY lower(industry)`
	if useSnapshots {
		q = `SELECT DISTINCT industry FROM company_snapshots WHERE pipeline_run_id = (SELECT id FROM pipeline_runs ORDER BY started_at DESC, id DESC LIMIT 1) AND COALESCE(industry,'') != '' ORDER BY lower(industry)`
	}
	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func snapshotReadEnabled(db *sql.DB) (bool, error) {
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM company_snapshots`).Scan(&n); err != nil {
		// Table may not exist on older DBs.
		if strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return false, nil
		}
		return false, err
	}
	return n > 0, nil
}

func salesStatusFromAction(action string) string {
	switch strings.TrimSpace(action) {
	case "Contact":
		return "qualified"
	case "Research first":
		return "partial_data"
	case "Ignore":
		return "needs_manual_review"
	default:
		return ""
	}
}

func leadStatusFromAction(action string) string {
	switch strings.TrimSpace(action) {
	case "Contact":
		return "new"
	case "Research first":
		return "needs_review"
	case "Ignore":
		return "discarded"
	default:
		return ""
	}
}

func icpMatchFromScore(score int) string {
	switch {
	case score >= 80:
		return "high"
	case score >= 50:
		return "medium"
	default:
		return "low"
	}
}
