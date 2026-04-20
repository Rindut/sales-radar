package pipeline

import (
	"salesradar/internal/discovery"
	"salesradar/internal/domain"
)

type RunOutcome string

const (
	RunOutcomeSuccess        RunOutcome = "success"
	RunOutcomePartialSuccess RunOutcome = "partial_success"
	RunOutcomeError          RunOutcome = "error"
)

func decideRunOutcome(stats RunStats, toggles domain.DiscoverySourceToggles) RunOutcome {
	// Core pipeline failed before producing run stats/rows.
	if stats.CandidatesFound == 0 && stats.RowsStored == 0 {
		return RunOutcomeError
	}
	if !toggles.WebsiteCrawl {
		return RunOutcomeSuccess
	}
	var web *discovery.ProviderStatus
	for i := range stats.ProviderStatuses {
		if stats.ProviderStatuses[i].ProviderName == "website_crawl_discovery" {
			web = &stats.ProviderStatuses[i]
			break
		}
	}
	if web == nil {
		return RunOutcomePartialSuccess
	}
	if web.Configured != nil && !*web.Configured {
		return RunOutcomePartialSuccess
	}
	switch web.State {
	case discovery.ProviderFailed, discovery.ProviderNotConfigured, discovery.ProviderDegraded:
		return RunOutcomePartialSuccess
	case discovery.ProviderDisabled:
		// Disabled in settings is non-degraded success; all other skip causes are partial.
		if web.ReasonCode == "disabled_by_settings" {
			return RunOutcomeSuccess
		}
		return RunOutcomePartialSuccess
	case discovery.ProviderSkipped:
		return RunOutcomePartialSuccess
	case discovery.ProviderSuccess:
		if web.CandidatesFailed > 0 || web.BudgetRowsSkipped > 0 {
			return RunOutcomePartialSuccess
		}
		if web.PagesAttempted > 0 && web.PagesSucceeded == 0 {
			return RunOutcomePartialSuccess
		}
	}
	return RunOutcomeSuccess
}
