// Package api registers JSON /api/v1 routes for Sales Radar (shared by cmd/api and cmd/web).
package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"salesradar/internal/api/debugview"
	"salesradar/internal/api/dto"
	"salesradar/internal/api/exportcsv"
	"salesradar/internal/api/jsonerr"
	"salesradar/internal/api/request"
	"salesradar/internal/apollo"
	"salesradar/internal/discovery"
	"salesradar/internal/icp"
	"salesradar/internal/pipeline"
	"salesradar/internal/store"
)

// Server exposes /health and /api/v1/* JSON handlers.
type Server struct {
	DB *sql.DB
}

// Register mounts API routes on mux. Safe to call alongside legacy HTML routes.
func Register(mux *http.ServeMux, db *sql.DB) {
	s := &Server{DB: db}
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /api/v1/leads", s.handleListLeads)
	mux.HandleFunc("GET /api/v1/leads/{id}", s.handleGetLead)
	mux.HandleFunc("GET /api/v1/settings", s.handleGetSettings)
	mux.HandleFunc("PUT /api/v1/settings", s.handlePutSettings)
	mux.HandleFunc("POST /api/v1/pipeline/run", s.handlePipelineRun)
	mux.HandleFunc("GET /api/v1/debug", s.handleDebug)
	mux.HandleFunc("GET /api/v1/export.csv", s.handleExportCSV)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	_ = enc.Encode(v)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "sales-radar-api"})
}

func (s *Server) handleListLeads(w http.ResponseWriter, r *http.Request) {
	f := request.ParseListFilter(r)
	leads, err := store.List(s.DB, f)
	if err != nil {
		jsonerr.Write(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	industries, _ := store.DistinctIndustries(s.DB)
	_, prErr := store.LatestPipelineRun(s.DB)
	if prErr != nil && !errors.Is(prErr, sql.ErrNoRows) {
		jsonerr.Write(w, http.StatusInternalServerError, "internal_error", prErr.Error())
		return
	}
	pipelineHasRun := prErr == nil
	totalInDB, err := store.Count(s.DB)
	if err != nil {
		jsonerr.Write(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	items := make([]dto.Lead, 0, len(leads))
	for _, l := range leads {
		items = append(items, dto.LeadFromStore(l))
	}
	q := r.URL.Query()
	resp := dto.LeadsListResponse{
		Items: items,
		Pagination: dto.Pagination{
			Total:    len(items),
			Returned: len(items),
		},
		Summary: dto.ListSummaryOptional{
			LastRun: dto.OptionalPipelineSummaryFromQuery(q),
		},
		Meta: dto.ListMeta{
			PipelineHasRun: pipelineHasRun,
			TotalInDB:      totalInDB,
			Industries:     industries,
		},
		FilterEcho: filterEcho(f),
	}
	writeJSON(w, http.StatusOK, resp)
}

func filterEcho(f store.ListFilter) map[string]string {
	order := "asc"
	if !f.OrderAsc {
		order = "desc"
	}
	return map[string]string{
		"q":            f.Query,
		"icp_match":    f.ICPMatch,
		"lead_status":  f.LeadStatus,
		"sales_status": f.SalesStatus,
		"industry":     f.Industry,
		"action":       f.Action,
		"sort":         f.SortBy,
		"order":        order,
	}
}

func (s *Server) handleGetLead(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimSpace(r.PathValue("id"))
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		jsonerr.Write(w, http.StatusNotFound, "not_found", "lead not found")
		return
	}
	lead, err := store.Get(s.DB, id)
	if errors.Is(err, sql.ErrNoRows) {
		jsonerr.Write(w, http.StatusNotFound, "not_found", "lead not found")
		return
	}
	if err != nil {
		jsonerr.Write(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, dto.LeadResponse{Lead: dto.LeadFromStore(lead)})
}

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	toggles, err := store.GetDiscoverySourceToggles(s.DB)
	if err != nil {
		jsonerr.Write(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	icpForm, err := store.GetICPFormSettings(s.DB)
	if err != nil {
		jsonerr.Write(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	resp := dto.SettingsResponse{
		DiscoverySources: dto.DiscoveryFromDomain(toggles),
		ICP:              dto.ICPFromStore(icpForm),
		Catalogs:         buildSettingsCatalogs(),
	}
	writeJSON(w, http.StatusOK, resp)
}

func buildSettingsCatalogs() dto.SettingsCatalogs {
	ind := icp.CatalogIndustries()
	outInd := make([]dto.CatalogOption, 0, len(ind))
	for _, o := range ind {
		outInd = append(outInd, dto.CatalogOption{ID: o.ID, Label: o.Label, Helper: o.Helper})
	}
	sig := icp.CatalogSignals()
	outSig := make([]dto.SignalCatalogOption, 0, len(sig))
	for _, o := range sig {
		outSig = append(outSig, dto.SignalCatalogOption{ID: o.ID, Label: o.Label, Helper: o.Helper, Keywords: append([]string(nil), o.Keywords...)})
	}
	reg := icp.CatalogRegions()
	outReg := make([]dto.CatalogOption, 0, len(reg))
	for _, o := range reg {
		outReg = append(outReg, dto.CatalogOption{ID: o.ID, Label: o.Label, Helper: o.Helper})
	}
	return dto.SettingsCatalogs{
		Industries: outInd,
		Signals:    outSig,
		Regions:    outReg,
		Weights:    append([]string(nil), icp.CatalogWeights()...),
	}
}

func (s *Server) handlePutSettings(w http.ResponseWriter, r *http.Request) {
	var body dto.PutSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonerr.Write(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}
	if err := store.SetDiscoverySourceToggles(s.DB, dto.DiscoveryToDomain(body.DiscoverySources)); err != nil {
		jsonerr.Write(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	icpStore := dto.ICPToStore(body.ICP)
	if err := store.SetICPFormSettings(s.DB, icpStore); err != nil {
		jsonerr.Write(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	togglesOut, err := store.GetDiscoverySourceToggles(s.DB)
	if err != nil {
		jsonerr.Write(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	icpOut, err := store.GetICPFormSettings(s.DB)
	if err != nil {
		jsonerr.Write(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, dto.SettingsResponse{
		DiscoverySources: dto.DiscoveryFromDomain(togglesOut),
		ICP:              dto.ICPFromStore(icpOut),
		Catalogs:         buildSettingsCatalogs(),
	})
}

func (s *Server) handlePipelineRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	params := pipeline.DefaultRunParams()
	toggles, err := store.GetDiscoverySourceToggles(s.DB)
	if err != nil {
		jsonerr.Write(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	params.SourceToggles = &toggles
	icpSaved, err := store.GetICPFormSettings(s.DB)
	if err != nil {
		jsonerr.Write(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	params.ICPRuntime = icpSaved.ToICPRuntime()
	prepared, stats, err := pipeline.RunWithQualityGate(ctx, params)
	if err != nil {
		jsonerr.Write(w, http.StatusInternalServerError, "pipeline_error", err.Error())
		return
	}
	inputs := make([]store.LeadInput, 0, len(prepared))
	for _, p := range prepared {
		inputs = append(inputs, store.FromStaged(p.Staged, p.Review))
	}
	runDebugJSON, _ := json.Marshal(stats)
	stored, err := store.ReplaceAll(s.DB, inputs, string(runDebugJSON))
	if err != nil {
		jsonerr.Write(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	rec, err := store.LatestPipelineRun(s.DB)
	if err != nil {
		jsonerr.Write(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	runInfo := dto.PipelineRunInfo{
		RunUUID:       rec.RunUUID,
		StartedAt:     rec.StartedAt,
		Status:        rec.Status,
		DiscoveryMode: rec.DiscoveryMode,
	}
	if rec.FinishedAt.Valid {
		runInfo.FinishedAt = rec.FinishedAt.String
	}
	resp := dto.PipelineRunResponse{
		Run:              runInfo,
		Stats:            stats,
		ProviderStatuses: append([]discovery.ProviderStatus(nil), stats.ProviderStatuses...),
		RowsPersisted:    stored,
	}
	// rows_stored in stats may differ naming — align with persisted count
	resp.Stats.RowsStored = stored
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleDebug(w http.ResponseWriter, r *http.Request) {
	rec, err := store.LatestPipelineRun(s.DB)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		jsonerr.Write(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	hasPersistedRun := err == nil
	var stats *pipeline.RunStats
	var statsDecodeErr string
	if hasPersistedRun && rec.RunDebugJSON.Valid && strings.TrimSpace(rec.RunDebugJSON.String) != "" {
		var st pipeline.RunStats
		if jerr := json.Unmarshal([]byte(rec.RunDebugJSON.String), &st); jerr != nil {
			statsDecodeErr = jerr.Error()
		} else {
			stats = &st
		}
	}
	hasFullDebug := stats != nil
	apolloOK := apollo.APIKeyFromEnv() != ""

	var summary dto.DebugSummary
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

	var providerRows []discovery.ProviderStatus
	var bdTotal string
	var bdOK bool
	discoveryMode := ""
	discoverySource := ""
	if hasFullDebug {
		providerRows = stats.ProviderStatuses
		bdTotal = fmt.Sprintf("%d", stats.BreakdownGeneratedTotal)
		bdOK = stats.BreakdownMatchesTotal
		discoveryMode = stats.DiscoveryMode
		discoverySource = stats.DiscoverySource
	}

	breakdownRows := debugview.BuildBreakdownRows(stats, apolloOK)
	apiBD := make([]dto.DebugBreakdownRow, 0, len(breakdownRows))
	for _, row := range breakdownRows {
		apiBD = append(apiBD, dto.DebugBreakdownRow{
			SourceName:       row.SourceName,
			Status:           row.Status,
			Generated:        row.Generated,
			Kept:             row.Kept,
			Qualified:        row.Qualified,
			Conversion:       row.Conversion,
			ConversionPct:    row.ConversionPct,
			SkipReason:       row.SkipReason,
			LastError:        row.LastError,
			IsError:          row.IsError,
			IsHighConversion: row.IsHighConversion,
		})
	}

	intRows := debugview.BuildIntegrationRows(hasPersistedRun, hasFullDebug, stats, apolloOK)
	apiInt := make([]dto.DebugIntegrationRow, 0, len(intRows))
	for _, row := range intRows {
		apiInt = append(apiInt, dto.DebugIntegrationRow{
			Host: row.Host, Role: row.Role, Config: row.Config, LastRun: row.LastRun,
		})
	}

	var runMeta *dto.DebugRunMeta
	if hasPersistedRun {
		id, uuid, started, finished, st, mode, hasDbg := debugview.FormatRunMeta(rec)
		runMeta = &dto.DebugRunMeta{
			RunID:         id,
			RunUUID:       uuid,
			StartedAt:     started,
			FinishedAt:    finished,
			Status:        st,
			DiscoveryMode: mode,
			HasDebugJSON:  hasDbg,
		}
	}

	out := dto.DebugResponse{
		Run:              runMeta,
		StatsDecodeError: statsDecodeErr,
		NoRunsInDB:       errors.Is(err, sql.ErrNoRows),
		HasPersistedRun:  hasPersistedRun,
		HasFullDebug:     hasFullDebug,
		Summary:          summary,
		ProviderRows:     providerRows,
		BreakdownRows:    apiBD,
		BreakdownTotal:   bdTotal,
		BreakdownOK:      bdOK,
		DiscoveryMode:    discoveryMode,
		DiscoverySource:  discoverySource,
		IntegrationRows:  apiInt,
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleExportCSV(w http.ResponseWriter, r *http.Request) {
	f := request.ParseListFilter(r)
	leads, err := store.List(s.DB, f)
	if err != nil {
		jsonerr.Write(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	exportcsv.WriteResponse(w, leads)
}
