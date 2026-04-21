package discovery

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"salesradar/internal/domain"
	"salesradar/internal/failpoint"
	"salesradar/internal/firecrawl"
)

func boolPtr(v bool) *bool { return &v }

// applyWebsiteEnrichmentToPool enriches each candidate with a non-empty official_domain in-place.
// Wall-clock limits: per-candidate timeout (Firecrawl + merge) and total budget across all rows so
// Generate Leads cannot hang on many sequential network calls.
func applyWebsiteEnrichmentToPool(ctx context.Context, p domain.RunParams, pool []domain.RawCandidate, websiteEnv bool) ([]domain.RawCandidate, []ProviderStatus) {
	tog := domain.SourceTogglesOrDefault(p.SourceToggles)
	cfg := firecrawl.Configured()
	enabledBySettings := tog.WebsiteCrawl
	st := ProviderStatus{
		ProviderName:      sourceWebsite,
		ProviderLabel:     "Firecrawl",
		Configured:        boolPtr(cfg),
		EnabledBySettings: boolPtr(enabledBySettings),
		BudgetLimitSec:    int(WebsiteEnrichTotalBudget().Seconds()),
	}
	switch {
	case !websiteEnv:
		st.State = ProviderDisabled
		st.SkipReason = "disabled by env (SALESRADAR_ENABLE_WEBSITE_CRAWL=0)"
		st.ReasonCode = "disabled_by_config"
		st.ReasonMessage = "Website crawl disabled by environment flag."
		slog.Info("website crawl: skipped", "reason", st.SkipReason)
		return pool, []ProviderStatus{st}
	case !tog.WebsiteCrawl:
		st.State = ProviderDisabled
		st.SkipReason = "disabled by settings"
		st.ReasonCode = "disabled_by_settings"
		st.ReasonMessage = "Website crawl disabled in discovery settings."
		slog.Info("website crawl: skipped", "reason", st.SkipReason)
		return pool, []ProviderStatus{st}
	case len(pool) == 0:
		st.State = ProviderSkipped
		st.SkipReason = "empty candidate pool"
		st.ReasonCode = "empty_candidate_pool"
		st.ReasonMessage = "No candidates available to enrich."
		return pool, []ProviderStatus{st}
	}
	if !cfg {
		st.State = ProviderNotConfigured
		st.ReasonCode = "provider_not_configured"
		st.ReasonMessage = "Website crawl is enabled but Firecrawl API key is not configured."
	}

	totalBudget := WebsiteEnrichTotalBudget()
	perCand := WebsiteEnrichPerCandidate()
	startedAt := time.Now()
	budgetEnd := time.Now().Add(totalBudget)
	slog.Info("website crawl: pass start",
		"pool", len(pool),
		"total_budget_sec", int(totalBudget.Seconds()),
		"per_candidate_sec", int(perCand.Seconds()),
	)

	var (
		noDomain       int
		fcSuccess      int
		legacy         int
		failed         int
		skipStatus     int
		budgetSkipped  int
		pagesAttempted int
		pagesSucceeded int
		reasonCounts   = map[string]int{}
	)

	out := make([]domain.RawCandidate, len(pool))
	for i := range pool {
		if time.Now().After(budgetEnd) {
			slog.Warn("website crawl: total time budget exhausted; rest copied without website enrichment",
				"remaining", len(pool)-i,
				"budget_sec", int(totalBudget.Seconds()),
			)
			for j := i; j < len(pool); j++ {
				out[j] = pool[j]
				budgetSkipped++
			}
			break
		}

		c := pool[i]
		if strings.TrimSpace(c.OfficialDomain) == "" {
			out[i] = c
			noDomain++
			continue
		}

		slog.Info("website crawl: candidate start", "i", i, "discovery_id", c.DiscoveryID, "host", c.OfficialDomain)
		t0 := time.Now()
		perCtx, cancel := context.WithTimeout(ctx, perCand)
		enriched := enrichWithWebsiteCrawl(perCtx, c, failpoint.WebsiteCrawl())
		deadlineHit := errors.Is(perCtx.Err(), context.DeadlineExceeded)
		cancel()
		elapsed := time.Since(t0)

		if len(enriched) == 0 {
			out[i] = c
			failed++
			slog.Warn("website crawl: enrich returned no candidates", "discovery_id", c.DiscoveryID, "elapsed_ms", elapsed.Milliseconds())
			continue
		}
		main := enriched[0]
		if !containsTrace(main.ProspectTrace.SourceTrace, domain.TraceWebsiteEnrichment) {
			main.ProspectTrace.SourceTrace = append(main.ProspectTrace.SourceTrace, domain.TraceWebsiteEnrichment)
		}
		out[i] = main

		if we := main.WebsiteEnrichment; we != nil {
			if rc := strings.TrimSpace(we.ReasonCode); rc != "" {
				reasonCounts[rc]++
			}
			pagesAttempted += we.PagesAttempted
			pagesSucceeded += we.PagesSucceeded
			switch strings.ToLower(strings.TrimSpace(we.Status)) {
			case "success":
				fcSuccess++
			case "legacy_fallback":
				legacy++
			case "skipped":
				skipStatus++
			default:
				failed++
			}
		} else {
			failed++
		}

		slog.Info("website crawl: candidate done",
			"i", i,
			"discovery_id", c.DiscoveryID,
			"elapsed_ms", elapsed.Milliseconds(),
			"deadline_hit", deadlineHit,
		)
	}

	withDomain := len(pool) - noDomain
	st.State = ProviderSuccess
	st.PagesAttempted = pagesAttempted
	st.PagesSucceeded = pagesSucceeded
	st.CandidatesTotal = len(pool)
	st.CandidatesSuccess = fcSuccess + legacy
	st.CandidatesSkipped = skipStatus + noDomain + budgetSkipped
	st.CandidatesFailed = failed
	st.BudgetRowsSkipped = budgetSkipped
	st.BudgetUsedSec = int(time.Since(startedAt).Seconds())
	st.Details = map[string]any{
		"pool":                len(pool),
		"with_domain":         withDomain,
		"firecrawl_success":   fcSuccess,
		"legacy_http":         legacy,
		"skipped_empty":       skipStatus,
		"other_or_failed":     failed,
		"no_domain":           noDomain,
		"budget_rows_skipped": budgetSkipped,
		"reason_counts":       reasonCounts,
	}

	if withDomain == 0 {
		st.State = ProviderSkipped
		st.SkipReason = "no candidates with official_domain"
		st.ReasonCode = "no_valid_domain"
		st.ReasonMessage = "No candidates had a valid official domain for crawl."
		slog.Info("website crawl: skipped", "reason", st.SkipReason, "pool_size", len(pool))
	} else {
		switch failpoint.WebsiteCrawl() {
		case failpoint.WebsiteCrawlSuccess:
			st.Configured = boolPtr(true)
			st.ReasonCode = "failpoint_website_success"
			st.ReasonMessage = "Website crawl success forced by verification failpoint."
		case failpoint.WebsiteCrawlTimeout:
			st.Configured = boolPtr(true)
			st.State = ProviderDegraded
			st.ReasonCode = "provider_timeout"
			st.ReasonMessage = "Website crawl timeout forced by verification failpoint."
		case failpoint.WebsiteCrawlError:
			st.Configured = boolPtr(true)
			st.State = ProviderFailed
			st.ReasonCode = "unknown_provider_error"
			st.ReasonMessage = "Website crawl error forced by verification failpoint."
		default:
			if !cfg {
				st.State = ProviderNotConfigured
				st.ReasonCode = "provider_not_configured"
				st.ReasonMessage = "Website crawl is enabled but Firecrawl API key is not configured."
			} else if failed > 0 && st.CandidatesSuccess == 0 {
				st.State = ProviderFailed
				st.ReasonCode = dominantReasonCode(reasonCounts, "unknown_provider_error")
				st.ReasonMessage = reasonCodeMessage(st.ReasonCode)
			} else if failed > 0 || budgetSkipped > 0 || (st.PagesAttempted > 0 && st.PagesSucceeded == 0) {
				st.State = ProviderDegraded
				if budgetSkipped > 0 {
					st.ReasonCode = "crawl_budget_exhausted"
					st.ReasonMessage = "Website crawl reached processing budget before finishing all candidates."
				} else {
					st.ReasonCode = dominantReasonCode(reasonCounts, "unknown_provider_error")
					st.ReasonMessage = reasonCodeMessage(st.ReasonCode)
				}
			}
		}
		slog.Info("website crawl: pass complete",
			"pool", len(pool),
			"with_domain", withDomain,
			"firecrawl_success", fcSuccess,
			"legacy_http", legacy,
			"no_domain", noDomain,
			"budget_rows_skipped", budgetSkipped,
		)
	}

	return out, []ProviderStatus{st}
}

func dominantReasonCode(reasonCounts map[string]int, fallback string) string {
	best := fallback
	bestN := 0
	for k, v := range reasonCounts {
		if v > bestN {
			best = k
			bestN = v
		}
	}
	return best
}

func reasonCodeMessage(code string) string {
	switch strings.TrimSpace(code) {
	case "provider_not_configured":
		return "Website crawl provider is not configured."
	case "provider_timeout":
		return "Website crawl timed out."
	case "provider_forbidden":
		return "Website crawl provider rejected request (forbidden)."
	case "provider_http_error":
		return "Website crawl provider returned an HTTP error."
	case "crawl_budget_exhausted":
		return "Website crawl budget exhausted before completion."
	case "no_valid_domain":
		return "No valid official domain was available."
	case "disabled_by_settings":
		return "Website crawl is disabled by settings."
	case "disabled_by_config":
		return "Website crawl is disabled by runtime config."
	default:
		return "Website crawl encountered an unknown provider error."
	}
}
