package icp

import (
	"strings"

	"salesradar/internal/companycheck"
	"salesradar/internal/domain"
	"salesradar/internal/enrich"
)

func weightMult(label string) float64 {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "high":
		return 1.2
	case "low":
		return 0.82
	default:
		return 1.0
	}
}

func regionScoreBump(lead *domain.ICPLead, focus string) int {
	if lead == nil || strings.TrimSpace(focus) == "" {
		return 0
	}
	lower := strings.ToLower(combinedContext(&lead.ExtractedLead))
	switch focus {
	case "idn":
		if containsAny(lower, "indonesia", "jakarta", "surabaya", "bandung", "yogyakarta") {
			return 4
		}
	case "sea":
		if containsAny(lower, "indonesia", "malaysia", "singapore", "thailand", "vietnam", "philippines", "jakarta", "bangkok") {
			return 4
		}
	}
	return 0
}

// applyICPScore sets ICPScore (0–100) and ScoreAction from match, bucket, size, LXP/domain/signals.
func applyICPScore(lead *domain.ICPLead, bucket domain.ICPIndustryBucket, sizeState sizeDecision, lxp bool, cfg *domain.ICPRuntimeSettings) {
	if lead == nil {
		return
	}
	if cfg == nil {
		cfg = domain.DefaultICPRuntimeSettings()
	}
	ctx := strings.ToLower(combinedContext(&lead.ExtractedLead))

	wI := weightMult(cfg.WeightIndustry)
	wSig := weightMult(cfg.WeightSignal)
	wSz := weightMult(cfg.WeightSize)

	score := 0
	switch lead.ICPMatch {
	case domain.ICPYes:
		score += 38
	case domain.ICPPartial:
		score += 26
	default:
		score += 6
	}

	var bucketPts float64
	switch bucket {
	case domain.BucketBanking:
		bucketPts = 18
	case domain.BucketRetail:
		bucketPts = 15
	case domain.BucketHospitality:
		bucketPts = 12
	}
	score += int(bucketPts * wI)

	var sizePts float64
	switch sizeState {
	case sizeMeets:
		sizePts = 22
	case sizeUnknown:
		sizePts = 8
	}
	score += int(sizePts * wSz)

	var lxpPts float64
	if lxp {
		lxpPts = 10
	}
	score += int(lxpPts * wSig)
	if strings.Contains(ctx, "growth_signal") {
		score += int(5 * wSig)
	}
	if strings.Contains(ctx, "job_signal") {
		score += int(4 * wSig)
	}

	score += regionScoreBump(lead, cfg.RegionFocus)

	dom := companycheck.SanitizeCompanyWebsiteDomain(lead.OfficialDomain)
	if dom == "" {
		dom = enrich.WebsiteDomainFromRef(lead.SourceRef)
	}
	dom = companycheck.SanitizeCompanyWebsiteDomain(dom)
	if dom != "" {
		score += 12
	}

	if lead.ProspectTrace.UsedApollo {
		score += 3
	}
	if lead.ProspectTrace.UsedLinkedIn {
		score += 3
	}

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
