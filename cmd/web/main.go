// Command web serves the internal Sales Radar UI (SQLite + HTML).
package main

import (
	"database/sql"
	"embed"
	"encoding/csv"
	"io/fs"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"salesradar/internal/apollo"
	"salesradar/internal/discovery"
	"salesradar/internal/domain"
	"salesradar/internal/googlesearch"
	"salesradar/internal/icp"
	"salesradar/internal/pipeline"
	"salesradar/internal/store"
)

//go:embed templates/*.html
var tmplFS embed.FS

//go:embed static
var embeddedStatic embed.FS

func listTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"initials":       initialsFromCompany,
		"readinessBadge": readinessBadgeHTML,
		"priorityPill":   priorityPillHTML,
		"signalPreview":  signalPreviewHTML,
		"actionLabel":    actionLabelHTML,
		"sourceLabel":    leadSourceLabelHTML,
		"add":            func(a, b int) int { return a + b },
		"icpListContains": func(list []string, v string) bool {
			for _, x := range list {
				if x == v {
					return true
				}
			}
			return false
		},
		"icpSub50On": func(p *bool) bool {
			if p == nil {
				return true
			}
			return *p
		},
		"suggestedAction":             suggestedActionFromLead,
		"drawerSuggestedLabelID":      drawerSuggestedLabelID,
		"suggestedActionReasonID":     suggestedActionReasonFromLeadID,
		"drawerHumanizeIcpReasonID":   drawerHumanizeIcpReasonID,
		"drawerTranslateNarrativeToID": drawerTranslateNarrativeToID,
		"drawerTranslateWhyNowToID":    drawerTranslateWhyNowToID,
		"drawerTranslateSalesAngleToID": drawerTranslateSalesAngleToID,
	}
}

func initialsFromCompany(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "?"
	}
	parts := strings.Fields(name)
	if len(parts) >= 2 {
		a, b := []rune(parts[0]), []rune(parts[1])
		if len(a) == 0 || len(b) == 0 {
			return "?"
		}
		return strings.ToUpper(string(a[0]) + string(b[0]))
	}
	runes := []rune(parts[0])
	if len(runes) >= 2 {
		return strings.ToUpper(string(runes[0]) + string(runes[1]))
	}
	if len(runes) == 1 {
		return strings.ToUpper(string(runes[0]))
	}
	return "?"
}

func truncateRunes(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || s == "" {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}

// readinessBadgeHTML maps pipeline state to sales-facing Ready / Almost ready / Not ready.
func readinessBadgeHTML(l store.Lead) template.HTML {
	if l.LeadStatus == "discarded" {
		return template.HTML(`<span class="badge-pill badge-readiness-not">Not ready</span>`)
	}
	switch strings.TrimSpace(l.Action) {
	case "Contact":
		return template.HTML(`<span class="badge-pill badge-readiness-ready">Ready</span>`)
	case "Research first":
		return template.HTML(`<span class="badge-pill badge-readiness-almost">Almost ready</span>`)
	case "Ignore":
		return template.HTML(`<span class="badge-pill badge-readiness-not">Not ready</span>`)
	}
	if l.SalesReady {
		return template.HTML(`<span class="badge-pill badge-readiness-ready">Ready</span>`)
	}
	switch l.SalesStatus {
	case "qualified":
		return template.HTML(`<span class="badge-pill badge-readiness-almost">Almost ready</span>`)
	case "partial_data", "needs_manual_review":
		return template.HTML(`<span class="badge-pill badge-readiness-almost">Almost ready</span>`)
	}
	return template.HTML(`<span class="badge-pill badge-readiness-not">Not ready</span>`)
}

func priorityPillHTML(score int, icp string) template.HTML {
	level := "low"
	label := "Low"
	if strings.EqualFold(icp, "high") || score >= 70 {
		level, label = "high", "High"
	} else if strings.EqualFold(icp, "medium") || score >= 40 {
		level, label = "medium", "Medium"
	}
	return template.HTML(`<span class="badge-pill badge-priority badge-priority-` + level + `">` + label + `</span>`)
}

func signalPreviewHTML(l store.Lead) template.HTML {
	s := strings.TrimSpace(l.WhyNow)
	if s == "" {
		switch strings.ToLower(strings.TrimSpace(l.WhyNowStrength)) {
		case "high":
			return template.HTML(`<span class="lc-signal lc-signal-fallback" title="No why-now text stored">Strong urgency</span>`)
		case "medium":
			return template.HTML(`<span class="lc-signal lc-signal-fallback" title="No why-now text stored">Moderate urgency</span>`)
		case "low":
			return template.HTML(`<span class="lc-signal lc-signal-empty">—</span>`)
		}
		return template.HTML(`<span class="lc-signal lc-signal-empty">—</span>`)
	}
	short := truncateRunes(s, 52)
	return template.HTML(`<span class="lc-signal" title="` + html.EscapeString(s) + `">` + html.EscapeString(short) + `</span>`)
}

func actionLabelHTML(l store.Lead) template.HTML {
	switch strings.TrimSpace(l.Action) {
	case "Contact":
		return template.HTML(`<span class="lc-action lc-action-contact">Contact now</span>`)
	case "Research first":
		return template.HTML(`<span class="lc-action lc-action-muted">Research first</span>`)
	case "Ignore":
		return template.HTML(`<span class="lc-action lc-action-muted">Ignore</span>`)
	}
	if strings.TrimSpace(l.Action) == "" {
		return template.HTML(`<span class="lc-action lc-action-muted">—</span>`)
	}
	return template.HTML(`<span class="lc-action lc-action-muted">` + html.EscapeString(l.Action) + `</span>`)
}

// friendlyLeadSource returns a short sales-facing label from trace or stored source enum.
func friendlyLeadSource(l store.Lead) string {
	for _, t := range l.SourceTrace {
		if s := discoveryTraceToLabel(strings.TrimSpace(t)); s != "" {
			return s
		}
	}
	return sourceEnumToLabel(strings.TrimSpace(l.Source))
}

func discoveryTraceToLabel(trace string) string {
	switch strings.ToLower(trace) {
	case "google_discovery":
		return "Google"
	case "seed_discovery":
		return "Seed"
	case "directory_discovery":
		return "Directory"
	case "website_crawl_discovery":
		return "Website crawl"
	case "job_signal_discovery":
		return "Job signal"
	case "mock_discovery":
		return "Mock"
	case "apollo_enrichment":
		return "Apollo"
	case "linkedin_validation", "linkedin_signal":
		return "LinkedIn"
	case "company_website_check":
		return ""
	default:
		return ""
	}
}

func sourceEnumToLabel(src string) string {
	switch strings.ToLower(src) {
	case "google":
		return "Google"
	case "linkedin":
		return "LinkedIn"
	case "apollo":
		return "Apollo"
	case "company_website":
		return "Company website"
	case "job_portal":
		return "Job portal"
	case "":
		return ""
	default:
		return src
	}
}

func leadSourceLabelHTML(l store.Lead) template.HTML {
	label := friendlyLeadSource(l)
	if label == "" {
		return template.HTML(`<span class="lc-source lc-source-empty">—</span>`)
	}
	traceTitle := strings.Join(l.SourceTrace, " → ")
	if traceTitle == "" {
		traceTitle = l.Source
	}
	return template.HTML(`<span class="lc-source" title="` + html.EscapeString(traceTitle) + `">` + html.EscapeString(label) + `</span>`)
}

// debugRunMeta is shown at the top of the Ops / Debug page (latest pipeline_runs row).
type debugRunMeta struct {
	RunID         int64
	RunUUID       string
	StartedAt     string
	FinishedAt    string
	Status        string
	DiscoveryMode string
	HasDebugJSON  bool
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

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbPath := flag.String("db", "data/salesradar.db", "SQLite database file path")
	flag.Parse()

	if err := os.MkdirAll(filepath.Dir(*dbPath), 0o755); err != nil {
		log.Fatal(err)
	}
	dsn := "file:" + *dbPath + "?_pragma=busy_timeout(5000)"
	db, err := store.Open(dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	tmpl := template.Must(template.New("").Funcs(listTemplateFuncs()).ParseFS(tmplFS, "templates/*.html"))

	staticFiles, err := fs.Sub(embeddedStatic, "static")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFiles))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/leads", http.StatusSeeOther)
	})
	mux.HandleFunc("/run", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ctx := r.Context()
		params := pipeline.DefaultRunParams()
		toggles, err := store.GetDiscoverySourceToggles(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		params.SourceToggles = &toggles
		icpSaved, err := store.GetICPFormSettings(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		params.ICPRuntime = icpSaved.ToICPRuntime()
		prepared, stats, err := pipeline.RunWithQualityGate(ctx, params)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		inputs := make([]store.LeadInput, 0, len(prepared))
		for _, p := range prepared {
			inputs = append(inputs, store.FromStaged(p.Staged, p.Review))
		}
		runDebugJSON, _ := json.Marshal(stats)
		stored, err := store.ReplaceAll(db, inputs, string(runDebugJSON))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if strings.Contains(r.Header.Get("Accept"), "application/json") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"candidates_found":          stats.CandidatesFound,
				"enriched":                  stats.Enriched,
				"contact_ready":             stats.ContactReady,
				"research_first":            stats.ResearchFirst,
				"rejected":                  stats.Rejected,
				"duplicates_removed":        stats.DuplicatesRemoved,
				"semantic_merged":           stats.SemanticMerged,
				"rows_stored":               stored,
				"integration_google_used":   stats.IntegrationGoogleUsed,
				"integration_apollo_used":   stats.IntegrationApolloUsed,
				"integration_linkedin_used": stats.IntegrationLinkedInUsed,
				"provider_statuses":         stats.ProviderStatuses,
				"discovery_mode":            stats.DiscoveryMode,
				"discovery_source":          stats.DiscoverySource,
				"discovery_breakdown":       stats.SourceBreakdown,
				"breakdown_generated_total": stats.BreakdownGeneratedTotal,
				"breakdown_matches_total":   stats.BreakdownMatchesTotal,
			})
			return
		}
		ig, ia, il := 0, 0, 0
		if stats.IntegrationGoogleUsed {
			ig = 1
		}
		if stats.IntegrationApolloUsed {
			ia = 1
		}
		if stats.IntegrationLinkedInUsed {
			il = 1
		}
		provJSON, _ := json.Marshal(stats.ProviderStatuses)
		breakdownJSON, _ := json.Marshal(stats.SourceBreakdown)
		q := fmt.Sprintf(
			"/leads?candidates=%d&enriched=%d&contact_ready=%d&research_first=%d&rejected=%d&dupes=%d&merged=%d&stored=%d&int_g=%d&int_a=%d&int_l=%d&providers=%s&breakdown=%s&bd_total=%d&bd_ok=%t&mode=%s&src=%s",
			stats.CandidatesFound, stats.Enriched, stats.ContactReady, stats.ResearchFirst, stats.Rejected, stats.DuplicatesRemoved, stats.SemanticMerged, stored, ig, ia, il,
			url.QueryEscape(string(provJSON)), url.QueryEscape(string(breakdownJSON)), stats.BreakdownGeneratedTotal, stats.BreakdownMatchesTotal, url.QueryEscape(stats.DiscoveryMode), url.QueryEscape(stats.DiscoverySource),
		)
		http.Redirect(w, r, q, http.StatusSeeOther)
	})
	mux.HandleFunc("/settings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if err := r.ParseForm(); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			t := domain.DiscoverySourceToggles{
				Google:       r.FormValue("google") == "on",
				Seed:         r.FormValue("seed") == "on",
				WebsiteCrawl: r.FormValue("website_crawl") == "on",
				JobSignal:    r.FormValue("job_signal") == "on",
				Apollo:       r.FormValue("apollo") == "on",
				LinkedIn:     r.FormValue("linkedin") == "on",
			}
			if err := store.SetDiscoverySourceToggles(db, t); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			sub50 := r.FormValue("apply_sub50") == "on"
			icp := store.ICPFormSettings{
				TargetIndustries:   dedupeFormValues(r.Form["icp_target_industry"]),
				RegionFocus:        strings.TrimSpace(r.FormValue("icp_region_focus")),
				SignalKeys:         dedupeFormValues(r.Form["icp_signal"]),
				ExcludedIndustries: nil,
				ExcludedSegments:   nil,
				ApplySub50:         &sub50,
				MinEmployees:        strings.TrimSpace(r.FormValue("icp_min_employees")),
				MaxEmployees:        strings.TrimSpace(r.FormValue("icp_max_employees")),
				WeightIndustry: strings.TrimSpace(r.FormValue("weight_industry")),
				WeightSignal:   strings.TrimSpace(r.FormValue("weight_signal")),
				WeightSize:     strings.TrimSpace(r.FormValue("weight_size")),
			}
			if err := store.SetICPFormSettings(db, icp); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			http.Redirect(w, r, "/settings", http.StatusSeeOther)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		toggles, err := store.GetDiscoverySourceToggles(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		icpForm, err := store.GetICPFormSettings(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data := struct {
			Rows              []discoverySourceRow
			ICP               store.ICPFormSettings
			CatalogIndustries []icp.IndustryOption
			CatalogSignals    []icp.SignalOption
			CatalogRegions    []icp.RegionOption
			CatalogWeights    []string
		}{
			Rows:              buildDiscoverySourceRows(toggles),
			ICP:               icpForm,
			CatalogIndustries: icp.CatalogIndustries(),
			CatalogSignals:    icp.CatalogSignals(),
			CatalogRegions:    icp.CatalogRegions(),
			CatalogWeights:    icp.CatalogWeights(),
		}
		_ = tmpl.ExecuteTemplate(w, "settings.html", data)
	})
	mux.HandleFunc("/leads", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		f := parseListFilter(r)
		leads, err := store.List(db, f)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		industries, _ := store.DistinctIndustries(db)
		qu := r.URL.Query()
		hasRun := qu.Get("candidates") != ""

		_, prErr := store.LatestPipelineRun(db)
		if prErr != nil && !errors.Is(prErr, sql.ErrNoRows) {
			http.Error(w, prErr.Error(), http.StatusInternalServerError)
			return
		}
		pipelineHasRun := prErr == nil
		totalLeadsInDB, err := store.Count(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		summary := struct {
			HasRun        bool
			TotalLeads    string
			ContactReady  string
			PendingReview string
			Candidates    string
			Enriched      string
			Rejected      string
			Duplicates    string
			Merged        string
			PipelineText  string
		}{
			HasRun:        hasRun,
			TotalLeads:    qu.Get("stored"),
			ContactReady:  qu.Get("contact_ready"),
			PendingReview: qu.Get("research_first"),
			Candidates:    qu.Get("candidates"),
			Enriched:      qu.Get("enriched"),
			Rejected:      qu.Get("rejected"),
			Duplicates:    qu.Get("dupes"),
			Merged:        qu.Get("merged"),
			PipelineText: fmt.Sprintf(
				"Pipeline finished — candidates: %s · enriched: %s · contact-ready qualified: %s · research-first: %s · rejected: %s · dupes removed: %s · semantic merged: %s · rows stored: %s",
				qu.Get("candidates"), qu.Get("enriched"), qu.Get("contact_ready"), qu.Get("research_first"), qu.Get("rejected"), qu.Get("dupes"), qu.Get("merged"), qu.Get("stored"),
			),
		}
		queryString := r.URL.RawQuery
		preserveParams := preserveRunQueryParams(qu)
		data := struct {
			Leads           []store.Lead
			Filter          store.ListFilter
			Summary         struct {
				HasRun        bool
				TotalLeads    string
				ContactReady  string
				PendingReview string
				Candidates    string
				Enriched      string
				Rejected      string
				Duplicates    string
				Merged        string
				PipelineText  string
			}
			PipelineHasRun bool
			TotalLeadsInDB int
			Total          int
			Industries     []string
			QueryString    string
			PreserveParams []queryParam
		}{
			Leads:           leads,
			Filter:          f,
			Summary:         summary,
			PipelineHasRun:  pipelineHasRun,
			TotalLeadsInDB:  totalLeadsInDB,
			Total:           len(leads),
			Industries:      industries,
			QueryString:     queryString,
			PreserveParams:  preserveParams,
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = tmpl.ExecuteTemplate(w, "list.html", data)
	})
	mux.HandleFunc("/debug", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		rec, err := store.LatestPipelineRun(db)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		hasPersistedRun := err == nil
		var stats *pipeline.RunStats
		var statsDecodeErr string
		if hasPersistedRun && rec.RunDebugJSON.Valid && strings.TrimSpace(rec.RunDebugJSON.String) != "" {
			var s pipeline.RunStats
			if jerr := json.Unmarshal([]byte(rec.RunDebugJSON.String), &s); jerr != nil {
				statsDecodeErr = jerr.Error()
			} else {
				stats = &s
			}
		}
		hasFullDebug := stats != nil
		summary := struct {
			HasRun        bool
			TotalLeads    string
			ContactReady  string
			PendingReview string
			Candidates    string
			Enriched      string
			Rejected      string
			Duplicates    string
			Merged        string
			PipelineText  string
		}{}
		if hasFullDebug {
			summary.HasRun = true
			summary.TotalLeads = fmt.Sprintf("%d", stats.RowsStored)
			summary.ContactReady = fmt.Sprintf("%d", stats.ContactReady)
			summary.PendingReview = fmt.Sprintf("%d", stats.ResearchFirst)
			summary.Candidates = fmt.Sprintf("%d", stats.CandidatesFound)
			summary.Enriched = fmt.Sprintf("%d", stats.Enriched)
			summary.Rejected = fmt.Sprintf("%d", stats.Rejected)
			summary.Duplicates = fmt.Sprintf("%d", stats.DuplicatesRemoved)
			summary.Merged = fmt.Sprintf("%d", stats.SemanticMerged)
			summary.PipelineText = fmt.Sprintf(
				"Pipeline finished — candidates: %d · enriched: %d · contact-ready qualified: %d · research-first: %d · rejected: %d · dupes removed: %d · semantic merged: %d · rows stored: %d",
				stats.CandidatesFound, stats.Enriched, stats.ContactReady, stats.ResearchFirst, stats.Rejected, stats.DuplicatesRemoved, stats.SemanticMerged, stats.RowsStored,
			)
		}
		googleOK := googlesearch.ConfigFromEnv().Configured()
		apolloOK := apollo.APIKeyFromEnv() != ""
		var intG, intA, intL bool
		if hasFullDebug {
			intG, intA, intL = stats.IntegrationGoogleUsed, stats.IntegrationApolloUsed, stats.IntegrationLinkedInUsed
		}
		integrationRows := []struct {
			Host    string
			Role    string
			Config  string
			LastRun string
		}{
			{
				Host:    "google.com",
				Role:    "Google Custom Search API (live discovery — not treated as a company domain)",
				Config:  integrationConfigLine(googleOK, "SALESRADAR_GOOGLE_API_KEY + SALESRADAR_GOOGLE_CX set"),
				LastRun: integrationLastRunLine(hasPersistedRun, hasFullDebug, intG, googleOK, "live discovery ran"),
			},
			{
				Host:    "apollo.io",
				Role:    "Apollo API (enrichment by domain — never used as official_domain)",
				Config:  integrationConfigLine(apolloOK, "SALESRADAR_APOLLO_API_KEY set"),
				LastRun: integrationLastRunLine(hasPersistedRun, hasFullDebug, intA, apolloOK, "enrichment called for ≥1 candidate"),
			},
			{
				Host:    "linkedin.com",
				Role:    "LinkedIn company URL (from Apollo when available — not a primary discovery domain)",
				Config:  "N/A (no site-wide crawl; URLs validated when Apollo returns them)",
				LastRun: linkedinIntegrationLastRun(hasPersistedRun, hasFullDebug, intL, apolloOK),
			},
		}
		var providerRows []discovery.ProviderStatus
		var breakdownRows []discoveryDebugRow
		bdTotal := ""
		bdOK := false
		discoveryMode := ""
		discoverySource := ""
		if hasFullDebug {
			providerRows = stats.ProviderStatuses
			breakdownRows = buildDiscoveryDebugRows(
				sourceBreakdownFromPipeline(stats.SourceBreakdown),
				providerRows,
				true,
				apolloOK,
			)
			bdTotal = fmt.Sprintf("%d", stats.BreakdownGeneratedTotal)
			bdOK = stats.BreakdownMatchesTotal
			discoveryMode = stats.DiscoveryMode
			discoverySource = stats.DiscoverySource
		}
		var runMeta *debugRunMeta
		if hasPersistedRun {
			runMeta = &debugRunMeta{
				RunID:         rec.ID,
				RunUUID:       rec.RunUUID,
				StartedAt:     formatRunTime(rec.StartedAt),
				FinishedAt:    formatNullRunTime(rec.FinishedAt),
				Status:        rec.Status,
				DiscoveryMode: rec.DiscoveryMode,
				HasDebugJSON:  rec.RunDebugJSON.Valid && strings.TrimSpace(rec.RunDebugJSON.String) != "",
			}
		}
		data := struct {
			Summary         struct {
				HasRun        bool
				TotalLeads    string
				ContactReady  string
				PendingReview string
				Candidates    string
				Enriched      string
				Rejected      string
				Duplicates    string
				Merged        string
				PipelineText  string
			}
			RunMeta           *debugRunMeta
			StatsDecodeErr    string
			NoRunsInDB        bool
			HasPersistedRun   bool
			HasFullDebug      bool
			ProviderRows    []discovery.ProviderStatus
			BreakdownRows   []discoveryDebugRow
			BreakdownTotal  string
			BreakdownOK     bool
			DiscoveryMode   string
			DiscoverySource string
			DebugRows       []struct {
				Host    string
				Role    string
				Config  string
				LastRun string
			}
		}{
			Summary:         summary,
			RunMeta:         runMeta,
			StatsDecodeErr:  statsDecodeErr,
			NoRunsInDB:      errors.Is(err, sql.ErrNoRows),
			HasPersistedRun: hasPersistedRun,
			HasFullDebug:    hasFullDebug,
			ProviderRows:    providerRows,
			BreakdownRows:   breakdownRows,
			BreakdownTotal:  bdTotal,
			BreakdownOK:     bdOK,
			DiscoveryMode:   discoveryMode,
			DiscoverySource: discoverySource,
			DebugRows:       integrationRows,
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = tmpl.ExecuteTemplate(w, "debug.html", data)
	})
	mux.HandleFunc("/leads/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		idStr := strings.TrimPrefix(r.URL.Path, "/leads/")
		if idStr == "" || strings.Contains(idStr, "/") {
			http.NotFound(w, r)
			return
		}
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		lead, err := store.Get(db, id)
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = tmpl.ExecuteTemplate(w, "detail.html", struct{ Lead store.Lead }{Lead: lead})
	})
	mux.HandleFunc("/export.csv", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		leads, err := store.List(db, parseListFilter(r))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fn := fmt.Sprintf("leads_export_%s.csv", time.Now().UTC().Format("20060102"))
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fn))
		cw := csv.NewWriter(w)
		_ = cw.Write([]string{
			"company", "industry", "size", "employee_size", "official_domain", "website_domain", "country_region", "reason_for_fit", "why_now", "why_now_strength", "sales_angle", "priority_score", "data_completeness",
			"action", "sales_ready", "sales_status", "icp_match", "duplicate_status", "lead_status", "confidence", "summary", "accept_explanation",
			"reasons", "missing_optional", "source", "source_ref", "created_at",
			"source_trace", "used_google", "used_apollo", "used_linkedin",
		})
		for _, l := range leads {
			reasons := strings.Join(l.Reasons, " | ")
			miss := strings.Join(l.MissingOptional, " | ")
			sr := "false"
			if l.SalesReady {
				sr = "true"
			}
			trace := strings.Join(l.SourceTrace, " | ")
			ug, ua, ul := "false", "false", "false"
			if l.UsedGoogle {
				ug = "true"
			}
			if l.UsedApollo {
				ua = "true"
			}
			if l.UsedLinkedIn {
				ul = "true"
			}
			_ = cw.Write([]string{
				l.Company,
				l.Industry,
				l.Size,
				l.EmployeeSize,
				l.OfficialDomain,
				l.WebsiteDomain,
				l.CountryRegion,
				l.ReasonForFit,
				l.WhyNow,
				l.WhyNowStrength,
				l.SalesAngle,
				fmt.Sprintf("%d", l.PriorityScore),
				fmt.Sprintf("%d", l.DataCompleteness),
				l.Action,
				sr,
				l.SalesStatus,
				l.ICPMatch,
				l.DuplicateStatus,
				l.LeadStatus,
				l.Confidence,
				l.Summary,
				l.AcceptExplanation,
				reasons,
				miss,
				l.Source,
				l.SourceRef,
				l.CreatedAt.UTC().Format(time.RFC3339),
				trace,
				ug, ua, ul,
			})
		}
		cw.Flush()
	})

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "address already in use") {
			log.Fatalf("cannot listen on %s: port already in use. Stop the other app using this port, or run with a different port, e.g. --addr :8081\n(%v)", *addr, err)
		}
		log.Fatal(err)
	}
	logURL := *addr
	if strings.HasPrefix(logURL, ":") {
		logURL = "127.0.0.1" + logURL
	}
	log.Printf("Sales Radar UI listening on http://%s (db=%s)", logURL, *dbPath)
	log.Fatal(http.Serve(ln, mux))
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

func linkedinIntegrationLastRun(hasPersistedRun, hasFullDebug, usedThisRun, apolloOK bool) string {
	if !hasPersistedRun {
		return "— (no pipeline run in database yet)"
	}
	if !hasFullDebug {
		return "— (no persisted debug payload — run Generate Leads to capture integration usage)"
	}
	if !apolloOK {
		return "Done — skipped (set SALESRADAR_APOLLO_API_KEY to receive LinkedIn company URLs)"
	}
	if usedThisRun {
		return "Done — LinkedIn company URL from Apollo for ≥1 candidate"
	}
	return "Done — no LinkedIn URL returned by Apollo this run"
}

func decodeProviderStatuses(raw string) []discovery.ProviderStatus {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []discovery.ProviderStatus
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return []discovery.ProviderStatus{
			{
				ProviderName: "provider_status",
				State:        discovery.ProviderError,
				LastError:    "failed to decode provider status: " + err.Error(),
			},
		}
	}
	return out
}

type sourceBreakdownRow struct {
	SourceName string `json:"source_name"`
	Generated  int    `json:"generated"`
	Kept       int    `json:"kept"`
	Qualified  int    `json:"qualified"`
	Rejected   int    `json:"rejected"`
}

type discoveryDebugRow struct {
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

func decodeSourceBreakdown(raw string) []sourceBreakdownRow {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []sourceBreakdownRow
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return []sourceBreakdownRow{
			{SourceName: "breakdown_decode_error", Generated: 0, Kept: 0, Rejected: 0},
		}
	}
	return out
}

func buildDiscoveryDebugRows(
	breakdown []sourceBreakdownRow,
	providers []discovery.ProviderStatus,
	hasRun bool,
	apolloConfigured bool,
) []discoveryDebugRow {
	bySource := map[string]sourceBreakdownRow{}
	for _, r := range breakdown {
		bySource[r.SourceName] = r
	}
	// Canonical PRD order first; then any extra sources (apollo, linkedin, unknown) sorted.
	order := []string{
		"google_discovery",
		"seed_discovery",
		"directory_discovery",
		"website_crawl_discovery",
		"job_signal_discovery",
		"mock_discovery",
		"apollo_enrichment",
		"linkedin_signal",
	}
	providerByName := map[string]discovery.ProviderStatus{}
	for _, p := range providers {
		providerByName[p.ProviderName] = p
	}
	seen := map[string]struct{}{}
	rows := make([]discoveryDebugRow, 0, len(bySource)+len(order))
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
) discoveryDebugRow {
	status := "skipped"
	skipReason := ""
	lastErr := ""
	if p, ok := providerByName[src]; ok {
		status = string(p.State)
		skipReason = strings.TrimSpace(p.SkipReason)
		lastErr = strings.TrimSpace(p.LastError)
	} else if hasRun && (b.Generated > 0 || b.Kept > 0 || b.Qualified > 0) {
		status = "active"
	}
	if status == "active" {
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
	if status == "skipped" || status == "not_configured" {
		skipReason = coalesceText(skipReason, inferSkipReason(src, apolloConfigured))
		lastErr = "—"
	}
	if status == "error" {
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
	return discoveryDebugRow{
		SourceName:       src,
		Status:           status,
		Generated:        b.Generated,
		Kept:             b.Kept,
		Qualified:        b.Qualified,
		Conversion:       conv,
		ConversionPct:    convPct,
		SkipReason:       skipReason,
		LastError:        lastErr,
		IsError:          status == "error",
		IsHighConversion: b.Generated > 0 && convPct > 50,
	}
}

func inferSkipReason(source string, apolloConfigured bool) string {
	switch source {
	case "directory_discovery":
		return "no eligible candidates"
	case "apollo_enrichment":
		if !apolloConfigured {
			return "missing API key"
		}
		return "provider not implemented"
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

func dedupeFormValues(values []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range values {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

type queryParam struct {
	Key   string
	Value string
}

func preserveRunQueryParams(qu url.Values) []queryParam {
	keys := []string{
		"candidates", "enriched", "contact_ready", "research_first", "rejected", "dupes", "merged", "stored",
		"int_g", "int_a", "int_l",
		"providers", "breakdown", "bd_total", "bd_ok", "mode", "src",
	}
	var out []queryParam
	for _, k := range keys {
		if v := strings.TrimSpace(qu.Get(k)); v != "" {
			out = append(out, queryParam{Key: k, Value: qu.Get(k)})
		}
	}
	return out
}

type discoverySourceRow struct {
	Key         string
	Name        string
	Description string
	Enabled     bool
	Badge       string
	Dependency  string
	UsageHint   string
	LastError   string
	FormName    string
}

func buildDiscoverySourceRows(t domain.DiscoverySourceToggles) []discoverySourceRow {
	gCfg := googlesearch.ConfigFromEnv()
	apolloOK := strings.TrimSpace(apollo.APIKeyFromEnv()) != ""
	return []discoverySourceRow{
		{
			Key: "google", Name: "Google", FormName: "google",
			Description: "Web search to surface candidate companies.",
			Dependency:  "Requires Google Custom Search API key and Search Engine ID (CX).",
			UsageHint:   "Availability can depend on how your Google Cloud project is set up.",
			Enabled:     t.Google,
			Badge:       badgeGoogle(gCfg.Configured(), t.Google),
		},
		{
			Key: "seed", Name: "Seed Discovery", FormName: "seed",
			Description: "Built-in list of distinct companies.",
			Dependency:  "Works without an external API.",
			UsageHint:   "Good baseline source.",
			Enabled:     t.Seed,
			Badge:       badgeSeed(t.Seed),
		},
		{
			Key: "website_crawl", Name: "Website Crawl", FormName: "website_crawl",
			Description: "Pulls extra context from company websites after a domain is known.",
			Dependency:  "Needs a valid company website domain.",
			UsageHint:   "Best used after discovery has found domains.",
			Enabled:     t.WebsiteCrawl,
			Badge:       badgeStandard(t.WebsiteCrawl),
		},
		{
			Key: "job_signal", Name: "Job Signal", FormName: "job_signal",
			Description: "Surfaces hiring and training-related signals from job-style clues.",
			Dependency:  "Works best when a company name or domain is already in context.",
			UsageHint:   "Best used after initial discovery.",
			Enabled:     t.JobSignal,
			Badge:       badgeStandard(t.JobSignal),
		},
		{
			Key: "apollo", Name: "Apollo", FormName: "apollo",
			Description: "Company enrichment and firmographics by domain.",
			Dependency:  "Requires an Apollo API key.",
			UsageHint:   "Adds detail after a domain exists—not the main way domains are found.",
			Enabled:     t.Apollo,
			Badge:       badgeApollo(apolloOK, t.Apollo),
		},
		{
			Key: "linkedin", Name: "LinkedIn", FormName: "linkedin",
			Description: "Adds company LinkedIn URLs when enrichment returns them.",
			Dependency:  "Depends on Apollo (or another step that yields a usable company URL).",
			UsageHint:   "Not a public crawl of LinkedIn.",
			Enabled:     t.LinkedIn,
			Badge:       badgeLinkedIn(apolloOK, t.Apollo, t.LinkedIn),
		},
	}
}

func badgeGoogle(configured, enabled bool) string {
	if !enabled {
		return "Disabled"
	}
	if !configured {
		return "Missing config"
	}
	return "Configured"
}

func badgeSeed(enabled bool) string {
	if !enabled {
		return "Disabled"
	}
	return "Enabled"
}

func badgeStandard(enabled bool) string {
	if !enabled {
		return "Disabled"
	}
	return "Enabled"
}

func badgeApollo(hasKey, enabled bool) string {
	if !enabled {
		return "Disabled"
	}
	if !hasKey {
		return "Missing config"
	}
	return "Configured"
}

func badgeLinkedIn(apolloKeyOK, apolloEnabled, enabled bool) string {
	if !enabled {
		return "Disabled"
	}
	if !apolloEnabled || !apolloKeyOK {
		return "Missing config"
	}
	return "Configured"
}

func parseListFilter(r *http.Request) store.ListFilter {
	q := r.URL.Query()
	f := store.ListFilter{
		Query:       strings.TrimSpace(q.Get("q")),
		ICPMatch:    q.Get("icp_match"),
		LeadStatus:  q.Get("lead_status"),
		SalesStatus: q.Get("sales_status"),
		Industry:    strings.TrimSpace(q.Get("industry")),
		Action:      strings.TrimSpace(q.Get("action")),
		SortBy:      q.Get("sort"),
		OrderAsc:    strings.ToLower(q.Get("order")) != "desc",
	}
	if f.SortBy != "priority" && f.SortBy != "confidence" && f.SortBy != "completeness" && f.SortBy != "action" && f.SortBy != "company" {
		f.SortBy = "priority"
	}
	return f
}
