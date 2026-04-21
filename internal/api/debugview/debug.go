// Package debugview builds the discovery/integration breakdown rows for GET /api/v1/debug.
// Logic mirrors cmd/web main debug handler (kept in sync intentionally).
package debugview

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"database/sql"

	"salesradar/internal/discovery"
	"salesradar/internal/googlesearch"
	"salesradar/internal/pipeline"
	"salesradar/internal/store"
)

// IntegrationRow is a row in the integration table.
type IntegrationRow struct {
	Host    string
	Role    string
	Config  string
	LastRun string
}

// BreakdownRow mirrors the HTML debug breakdown table.
type BreakdownRow struct {
	SourceName       string
	Status           string
	Generated        int
	Kept             int
	Qualified        int
	Conversion       string
	ConversionPct    float64
	SkipReason       string
	LastError        string
	IsError          bool
	IsHighConversion bool
}

type sourceBreakdownRow struct {
	SourceName string `json:"source_name"`
	Generated  int    `json:"generated"`
	Kept       int    `json:"kept"`
	Qualified  int    `json:"qualified"`
	Rejected   int    `json:"rejected"`
}

func sourceBreakdownFromPipeline(rows []pipeline.SourceBreakdown) []sourceBreakdownRow {
	out := make([]sourceBreakdownRow, len(rows))
	for i, r := range rows {
		out[i] = sourceBreakdownRow{
			SourceName: r.SourceName,
			Generated:  r.Generated,
			Kept:       r.Kept,
			Qualified:  r.Qualified,
			Rejected:   r.Rejected,
		}
	}
	return out
}

func formatRunTime(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "—"
	}
	t, e := time.Parse(time.RFC3339, s)
	if e != nil {
		return s
	}
	return t.Format("2006-01-02 15:04:05 MST")
}

func formatNullRunTime(ns sql.NullString) string {
	if !ns.Valid {
		return "—"
	}
	return formatRunTime(ns.String)
}

func integrationConfigLine(ok bool, detail string) string {
	if ok {
		return "Ready — " + detail
	}
	return "Not configured — " + detail
}

func integrationLastRunLine(hasPersistedRun, hasFullDebug, usedThisRun, canUse bool, whenUsed string) string {
	if !hasPersistedRun {
		return "— (no pipeline run in database yet)"
	}
	if !hasFullDebug {
		return "— (no persisted debug payload — run Generate Leads to capture integration usage)"
	}
	if !canUse {
		return "Done — skipped (integration not configured)"
	}
	if usedThisRun {
		return "Done — " + whenUsed
	}
	return "Done — not used this run (no candidate touched this integration)"
}

func linkedinIntegrationLastRun(hasPersistedRun, hasFullDebug, usedThisRun, linkedinOK bool) string {
	if !hasPersistedRun {
		return "— (no pipeline run in database yet)"
	}
	if !hasFullDebug {
		return "— (no persisted debug payload — run Generate Leads to capture integration usage)"
	}
	if !linkedinOK {
		return "Done — skipped (set LINKEDIN_API_KEY to enable LinkedIn integration)"
	}
	if usedThisRun {
		return "Done — LinkedIn company URL from Apollo for ≥1 candidate"
	}
	return "Done — no LinkedIn URL returned by Apollo this run"
}

// BuildIntegrationRows returns Google/Apollo/LinkedIn summary rows.
func BuildIntegrationRows(hasPersistedRun, hasFullDebug bool, stats *pipeline.RunStats, apolloOK, linkedinOK bool) []IntegrationRow {
	googleOK := googlesearch.ConfigFromEnv().Configured()
	var intG, intA, intL bool
	if hasFullDebug && stats != nil {
		intG, intA, intL = stats.IntegrationGoogleUsed, stats.IntegrationApolloUsed, stats.IntegrationLinkedInUsed
	}
	return []IntegrationRow{
		{
			Host:    "google.com",
			Role:    "Google Custom Search API (live discovery — not treated as a company domain)",
			Config:  integrationConfigLine(googleOK, "SALESRADAR_GOOGLE_API_KEY + SALESRADAR_GOOGLE_CX set"),
			LastRun: integrationLastRunLine(hasPersistedRun, hasFullDebug, intG, googleOK, "live discovery ran"),
		},
		{
			Host:    "apollo.io",
			Role:    "Apollo API (company discovery and optional domain enrichment)",
			Config:  integrationConfigLine(apolloOK, "APOLLO_API_KEY set"),
			LastRun: integrationLastRunLine(hasPersistedRun, hasFullDebug, intA, apolloOK, "discovery or enrichment touched ≥1 candidate"),
		},
		{
			Host:    "linkedin.com",
			Role:    "LinkedIn company URL (from Apollo when available — not a primary discovery domain)",
			Config:  integrationConfigLine(linkedinOK, "LINKEDIN_API_KEY set"),
			LastRun: linkedinIntegrationLastRun(hasPersistedRun, hasFullDebug, intL, linkedinOK),
		},
	}
}

func buildDiscoveryDebugRows(
	breakdown []sourceBreakdownRow,
	providers []discovery.ProviderStatus,
	hasRun bool,
	apolloConfigured bool,
) []BreakdownRow {
	bySource := map[string]sourceBreakdownRow{}
	for _, r := range breakdown {
		bySource[r.SourceName] = r
	}
	order := []string{
		"google_discovery",
		"seed_discovery",
		"directory_discovery",
		"website_crawl_discovery",
		"job_signal_discovery",
		"mock_discovery",
		"apollo_discovery",
		"linkedin_signal",
		"apollo_enrichment",
	}
	providerByName := map[string]discovery.ProviderStatus{}
	for _, p := range providers {
		providerByName[p.ProviderName] = p
	}
	seen := map[string]struct{}{}
	rows := make([]BreakdownRow, 0, len(bySource)+len(order))
	for _, src := range order {
		if _, ok := seen[src]; ok {
			continue
		}
		seen[src] = struct{}{}
		rows = append(rows, makeDiscoveryDebugRow(src, bySource[src], providerByName, hasRun, apolloConfigured))
	}
	var rest []string
	for src := range bySource {
		if _, ok := seen[src]; ok {
			continue
		}
		rest = append(rest, src)
	}
	sort.Strings(rest)
	for _, src := range rest {
		seen[src] = struct{}{}
		rows = append(rows, makeDiscoveryDebugRow(src, bySource[src], providerByName, hasRun, apolloConfigured))
	}
	return rows
}

func makeDiscoveryDebugRow(
	src string,
	b sourceBreakdownRow,
	providerByName map[string]discovery.ProviderStatus,
	hasRun bool,
	apolloConfigured bool,
) BreakdownRow {
	status := "skipped"
	skipReason := ""
	lastErr := ""
	if p, ok := providerByName[src]; ok {
		status = string(p.State)
		skipReason = strings.TrimSpace(p.SkipReason)
		lastErr = strings.TrimSpace(p.LastError)
		if src == "website_crawl_discovery" && b.Generated == 0 && (p.State == discovery.ProviderSuccess || p.State == discovery.ProviderDegraded) {
			status = "enrichment_only"
		}
	} else if hasRun && (b.Generated > 0 || b.Kept > 0 || b.Qualified > 0) {
		status = "success"
	}
	if status == "success" {
		if b.Generated == 0 {
			switch src {
			case "website_crawl_discovery":
				lastErr = coalesceText(lastErr, "executed with zero output: no website-qualified candidates")
			case "job_signal_discovery":
				lastErr = coalesceText(lastErr, "executed with zero output: no job-signal candidates")
			}
		}
		skipReason = "—"
	}
	if status == "skipped" || status == "not_configured" || status == "disabled" {
		skipReason = coalesceText(skipReason, inferSkipReason(src, apolloConfigured))
		lastErr = "—"
	}
	if status == "failed" || status == "degraded" {
		skipReason = "—"
	}
	conv := "N/A"
	convPct := 0.0
	if b.Generated > 0 {
		convPct = (float64(b.Qualified) / float64(b.Generated)) * 100
		conv = fmt.Sprintf("%d/%d (%.0f%%)", b.Qualified, b.Generated, convPct)
	}
	if lastErr == "" {
		lastErr = "—"
	}
	if skipReason == "" {
		skipReason = "—"
	}
	return BreakdownRow{
		SourceName:       src,
		Status:           status,
		Generated:        b.Generated,
		Kept:             b.Kept,
		Qualified:        b.Qualified,
		Conversion:       conv,
		ConversionPct:    convPct,
		SkipReason:       skipReason,
		LastError:        lastErr,
		IsError:          status == "failed" || status == "degraded",
		IsHighConversion: b.Generated > 0 && convPct > 50,
	}
}

func inferSkipReason(source string, apolloConfigured bool) string {
	switch source {
	case "directory_discovery":
		return "no eligible candidates"
	case "apollo_discovery":
		if !apolloConfigured {
			return "missing API key"
		}
		return "no Apollo companies returned"
	case "apollo_enrichment":
		if !apolloConfigured {
			return "missing API key"
		}
		return "upstream discovery did not require Apollo enrichment"
	case "linkedin_signal":
		if !apolloConfigured {
			return "dependency unavailable: Apollo missing API key"
		}
		return "provider not implemented"
	default:
		return "provider not implemented"
	}
}

func coalesceText(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}

// BuildBreakdownRows builds discovery breakdown rows from persisted stats.
func BuildBreakdownRows(stats *pipeline.RunStats, apolloOK bool) []BreakdownRow {
	if stats == nil {
		return nil
	}
	return buildDiscoveryDebugRows(
		sourceBreakdownFromPipeline(stats.SourceBreakdown),
		stats.ProviderStatuses,
		true,
		apolloOK,
	)
}

// FormatRunMeta formats pipeline run record fields for API.
func FormatRunMeta(rec store.PipelineRunRecord) (id int64, runUUID, started, finished, status, outcome, mode string, hasDbg bool) {
	return rec.ID, rec.RunUUID, formatRunTime(rec.StartedAt), formatNullRunTime(rec.FinishedAt), rec.Status, rec.RunOutcome, rec.DiscoveryMode,
		rec.RunDebugJSON.Valid && strings.TrimSpace(rec.RunDebugJSON.String) != ""
}
