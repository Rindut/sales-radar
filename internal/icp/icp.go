// Package icp applies business rules to produce ICP decisions for Sales Radar.
package icp

import (
	"context"
	"strings"

	"salesradar/internal/domain"
)

type sizeDecision int

const (
	sizeUnknown sizeDecision = iota
	sizeMeets
	sizeBelow
)

const reasonBelowSectorTypicalSize = "below typical company size for this sector"

// Evaluate assigns icp_match, icp_reason, and icp_industry_bucket.
// cfg nil uses domain.DefaultICPRuntimeSettings().
func Evaluate(ctx context.Context, e *domain.ExtractedLead, cfg *domain.ICPRuntimeSettings) (*domain.ICPLead, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if e == nil {
		return &domain.ICPLead{ICPMatch: domain.ICPNo, ICPScore: 0, ScoreAction: domain.ScoreActionReject}, nil
	}
	eff := cfg
	if eff == nil {
		eff = domain.DefaultICPRuntimeSettings()
	}

	bucket, plausible, ambiguous := resolveIndustryBucket(e)
	sizeState := evaluateSizeThreshold(bucket, e.CompanySizeEstimated)
	disq := applyDisqualifiers(e, bucket, plausible, sizeState, eff)

	match, reasons := decideMatch(e, bucket, plausible, ambiguous, sizeState, disq, eff)

	out := &domain.ICPLead{
		ExtractedLead: *e,
		ICPMatch:      match,
		ICPReason:     capReasons(reasons),
	}
	out.ICPIndustryBucket = bucket
	textCtx := combinedContext(e)
	lxp := lxpSignalsForKeys(textCtx, eff.SignalKeys)
	applyICPScore(out, bucket, sizeState, lxp, eff)
	return out, nil
}

func resolveIndustryBucket(e *domain.ExtractedLead) (bucket domain.ICPIndustryBucket, plausible bool, ambiguous bool) {
	if e.Industry != nil {
		s := strings.ToLower(strings.TrimSpace(*e.Industry))
		switch {
		case strings.Contains(s, "bank"):
			return domain.BucketBanking, true, false
		case strings.Contains(s, "retail"):
			return domain.BucketRetail, true, false
		case strings.Contains(s, "hospital") || strings.Contains(s, "hotel"):
			return domain.BucketHospitality, true, false
		default:
			return domain.BucketNone, false, false
		}
	}

	notes := ""
	if e.ExtractionNotes != nil {
		notes = strings.ToLower(*e.ExtractionNotes)
	}
	if strings.Contains(notes, "ambiguous industry") {
		return domain.BucketNone, true, true
	}
	if e.StrongClassification && e.UnstructuredContext == "" {
		return domain.BucketNone, true, false
	}
	return inferBucketFromContext(combinedContext(e))
}

func evaluateSizeThreshold(bucket domain.ICPIndustryBucket, size *string) sizeDecision {
	if size == nil || bucket == domain.BucketNone {
		return sizeUnknown
	}
	threshold := thresholdFor(bucket)
	if threshold == 0 {
		return sizeUnknown
	}

	text := normalizeNumericText(strings.ToLower(strings.TrimSpace(*size)))
	if text == "" {
		return sizeUnknown
	}

	if n, ok := parseOverValue(text); ok {
		if n >= threshold {
			return sizeMeets
		}
		// "over 800" does not prove >1000 for banking; keep conservative.
		return sizeUnknown
	}

	if lo, hi, ok := parseRange(text); ok {
		switch {
		case lo > threshold:
			return sizeMeets
		case hi <= threshold:
			return sizeBelow
		default:
			return sizeUnknown
		}
	}

	if n, ok := parseSingleNumber(text); ok {
		if n > threshold {
			return sizeMeets
		}
		return sizeBelow
	}
	return sizeUnknown
}

func bucketFromCatalogID(id string) domain.ICPIndustryBucket {
	switch strings.ToLower(strings.TrimSpace(id)) {
	case "banking":
		return domain.BucketBanking
	case "retail":
		return domain.BucketRetail
	case "hospitality":
		return domain.BucketHospitality
	default:
		return domain.BucketNone
	}
}

func industryTargetOk(cfg *domain.ICPRuntimeSettings, leadID string, bucket domain.ICPIndustryBucket) bool {
	if len(cfg.TargetIndustryIDs) == 0 {
		return true
	}
	if leadID != "" {
		for _, t := range cfg.TargetIndustryIDs {
			if strings.EqualFold(t, leadID) {
				return true
			}
		}
		return false
	}
	if bucket != domain.BucketNone {
		for _, t := range cfg.TargetIndustryIDs {
			if bucketFromCatalogID(t) == bucket {
				return true
			}
		}
		return false
	}
	return true
}

func industryExcluded(cfg *domain.ICPRuntimeSettings, leadID string, bucket domain.ICPIndustryBucket) bool {
	if leadID != "" {
		for _, x := range cfg.ExcludedIndustryIDs {
			if strings.EqualFold(x, leadID) {
				return true
			}
		}
	}
	if bucket != domain.BucketNone {
		for _, b := range cfg.ExcludedBuckets {
			if b == bucket {
				return true
			}
		}
	}
	return false
}

func applyDisqualifiers(e *domain.ExtractedLead, bucket domain.ICPIndustryBucket, plausible bool, sizeState sizeDecision, cfg *domain.ICPRuntimeSettings) []string {
	reasons := make([]string, 0, 8)
	ctx := combinedContext(e)
	leadID := InferLeadIndustryID(e, bucket)

	if industryExcluded(cfg, leadID, bucket) {
		reasons = appendReason(reasons, "non-target industry")
	}

	if !industryTargetOk(cfg, leadID, bucket) {
		reasons = appendReason(reasons, "non-target industry")
	}

	if sizeState == sizeBelow {
		reasons = appendReason(reasons, reasonBelowSectorTypicalSize)
	}

	if cfg.ApplySub50Rule && sizeClearlyBelowFifty(e.CompanySizeEstimated) {
		reasons = appendReason(reasons, "below 50 employees")
	}

	for _, d := range disqualifiersForConfiguredSize(e.CompanySizeEstimated, cfg.MinEmployees, cfg.MaxEmployees) {
		reasons = appendReason(reasons, d)
	}

	for _, d := range profileDisqualifiersSegments(ctx, cfg.ExcludedSegmentKeys) {
		reasons = appendReason(reasons, d)
	}

	notes := ""
	if e.ExtractionNotes != nil {
		notes = strings.ToLower(*e.ExtractionNotes)
	}
	if strings.Contains(notes, "no credible training relevance") || strings.Contains(notes, "not relevant") {
		reasons = appendReason(reasons, "no credible training relevance")
	}

	if !plausible && bucket == domain.BucketNone && e.CompanySizeEstimated == nil && e.Industry == nil && !e.StrongClassification {
		reasons = appendReason(reasons, "data too incomplete")
	}

	return reasons
}

func decideMatch(e *domain.ExtractedLead, bucket domain.ICPIndustryBucket, plausible bool, ambiguous bool, sizeState sizeDecision, disqualifiers []string, cfg *domain.ICPRuntimeSettings) (domain.ICPMatch, []string) {
	hardNo := []string{
		"non-target industry", reasonBelowSectorTypicalSize, "no credible training relevance",
		"below 50 employees", "disqualifying business profile",
		"below minimum company size", "above maximum company size",
	}
	for _, h := range hardNo {
		if hasDisqualifier(disqualifiers, h) {
			return domain.ICPNo, disqualifiers
		}
	}

	ctx := combinedContext(e)
	lxp := lxpSignalsForKeys(ctx, cfg.SignalKeys)

	// Strong yes: target sector, meets sector size rule, not ambiguous, clear training or ops signal.
	if bucket != domain.BucketNone && !ambiguous && sizeState == sizeMeets && lxp {
		reasons := []string{
			string(bucket) + " industry aligned with your targets",
			sizeReasonFor(bucket),
			"training or operational complexity signal in context",
		}
		return domain.ICPYes, reasons
	}

	// Partial: industry or size unclear, or incomplete signals.
	if plausible || ambiguous || bucket != domain.BucketNone {
		reasons := make([]string, 0, 2)
		if ambiguous {
			reasons = appendReason(reasons, "industry ambiguous across signals")
		} else if bucket != domain.BucketNone {
			reasons = appendReason(reasons, string(bucket)+" sector plausible")
		} else {
			reasons = appendReason(reasons, "signals incomplete versus your profile")
		}
		if sizeState == sizeUnknown {
			reasons = appendReason(reasons, "company size versus sector target is unclear")
		}
		if sizeState == sizeMeets && !lxp {
			reasons = appendReason(reasons, "meets size but weaker training signal in text")
		}
		if bucket != domain.BucketNone && sizeState == sizeUnknown && lxp {
			reasons = appendReason(reasons, "training signals without firm company size")
		}
		if len(reasons) == 0 {
			reasons = appendReason(reasons, "evidence incomplete")
		}
		return domain.ICPPartial, capReasons(reasons)
	}

	if len(disqualifiers) > 0 {
		return domain.ICPNo, disqualifiers
	}
	return domain.ICPNo, []string{"insufficient data for ICP evaluation"}
}

func thresholdFor(bucket domain.ICPIndustryBucket) int {
	switch bucket {
	case domain.BucketBanking:
		return 1000
	case domain.BucketRetail:
		return 200
	case domain.BucketHospitality:
		return 100
	default:
		return 0
	}
}

func sizeReasonFor(bucket domain.ICPIndustryBucket) string {
	switch bucket {
	case domain.BucketBanking:
		return "size appears above 1000 employees"
	case domain.BucketRetail:
		return "size appears above 200 employees"
	case domain.BucketHospitality:
		return "size appears above 100 employees"
	default:
		return "size appears above your sector target"
	}
}

func hasDisqualifier(reasons []string, needle string) bool {
	for _, r := range reasons {
		if strings.EqualFold(r, needle) {
			return true
		}
	}
	return false
}

func appendReason(reasons []string, reason string) []string {
	if reason == "" || len(reasons) >= domain.MaxICPReasons {
		return reasons
	}
	for _, existing := range reasons {
		if strings.EqualFold(existing, reason) {
			return reasons
		}
	}
	return append(reasons, reason)
}

func capReasons(reasons []string) []string {
	if len(reasons) <= domain.MaxICPReasons {
		return reasons
	}
	return reasons[:domain.MaxICPReasons]
}
