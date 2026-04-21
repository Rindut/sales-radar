package icp

import (
	"strings"

	"salesradar/internal/companycheck"
	"salesradar/internal/domain"
	"salesradar/internal/enrich"
)

func regionScoreBump(lead *domain.ICPLead, focus string) int {
	if lead == nil || strings.TrimSpace(focus) == "" {
		return 0
	}
	lower := strings.ToLower(combinedContext(&lead.ExtractedLead))
	switch focus {
	case "idn":
		if containsAny(lower, "indonesia", "jakarta", "surabaya", "bandung", "yogyakarta") {
			return 20
		}
	case "sea":
		if containsAny(lower, "indonesia", "malaysia", "singapore", "thailand", "vietnam", "philippines", "jakarta", "bangkok") {
			return 20
		}
	}
	return 0
}

func sizeScore(lead *domain.ICPLead, sizeState sizeDecision, cfg *domain.ICPRuntimeSettings) int {
	if lead == nil || cfg == nil {
		return 0
	}
	if sizeState == sizeMeets {
		return 30
	}
	if len(disqualifiersForConfiguredSize(lead.CompanySizeEstimated, cfg.MinEmployees, cfg.MaxEmployees)) == 0 {
		if cfg.MinEmployees > 0 || cfg.MaxEmployees > 0 {
			_, _, ok := settingsEmployeeBounds(lead.CompanySizeEstimated)
			if ok {
				return 30
			}
		}
	}
	return 0
}

// applyICPScore sets ICPScore (0–100) and ScoreAction from explicit ICP components.
func applyICPScore(lead *domain.ICPLead, bucket domain.ICPIndustryBucket, sizeState sizeDecision, lxp bool, cfg *domain.ICPRuntimeSettings) {
	if lead == nil {
		return
	}
	if cfg == nil {
		cfg = domain.DefaultICPRuntimeSettings()
	}
	leadID := InferLeadIndustryID(&lead.ExtractedLead, bucket)
	score := 0
	if industryTargetOk(cfg, leadID, bucket) && leadID != "" {
		score += 40
	}
	score += sizeScore(lead, sizeState, cfg)
	score += regionScoreBump(lead, cfg.RegionFocus)
	if lxp {
		score += 10
	}

	dom := companycheck.SanitizeCompanyWebsiteDomain(lead.OfficialDomain)
	if dom == "" {
		dom = enrich.WebsiteDomainFromRef(lead.SourceRef)
	}
	dom = companycheck.SanitizeCompanyWebsiteDomain(dom)
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	lead.ICPScore = score
	lead.ScoreAction = deriveScoreAction(lead, dom)
}

func deriveScoreAction(lead *domain.ICPLead, domainOK string) domain.ScoreAction {
	switch lead.ICPMatch {
	case domain.ICPNo:
		return domain.ScoreActionReject
	case domain.ICPPartial:
		return domain.ScoreActionResearch
	default: // yes
		if domainOK != "" && lead.ICPScore >= 68 {
			return domain.ScoreActionContact
		}
		return domain.ScoreActionResearch
	}
}
