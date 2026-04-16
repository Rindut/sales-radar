package icp

import (
	"strings"

	"salesradar/internal/domain"
)

func combinedContext(e *domain.ExtractedLead) string {
	var b strings.Builder
	if e.Industry != nil {
		b.WriteString(*e.Industry)
		b.WriteByte(' ')
	}
	b.WriteString(e.UnstructuredContext)
	if e.ExtractionNotes != nil {
		b.WriteString(" ")
		b.WriteString(*e.ExtractionNotes)
	}
	if e.CompanyName != nil {
		b.WriteString(" ")
		b.WriteString(*e.CompanyName)
	}
	return b.String()
}

func containsAny(lowerCtx string, words ...string) bool {
	for _, w := range words {
		if strings.Contains(lowerCtx, w) {
			return true
		}
	}
	return false
}

// inferBucketFromContext derives banking | retail | hospitality from free text (mock + extraction blurbs).
func inferBucketFromContext(ctx string) (bucket domain.ICPIndustryBucket, plausible bool, ambiguous bool) {
	lower := strings.ToLower(ctx)
	banking := containsAny(lower, "bank", "banking", "bumn", "private bank", "regulated", "branch network", "compliance-heavy")
	retail := containsAny(lower, "retail", "grocery", "supermarket", "department store", "outlet", "franchise", "f&b", "quick-service", "restaurant chain", "national grocery")
	hosp := containsAny(lower, "hospitality", "hotel", "housekeeping", "guest-facing", "boutique hotel")

	n := 0
	if banking {
		n++
	}
	if retail {
		n++
	}
	if hosp {
		n++
	}
	if n > 1 {
		return domain.BucketNone, true, true
	}
	if n == 0 {
		return domain.BucketNone, false, false
	}
	if banking {
		return domain.BucketBanking, true, false
	}
	if retail {
		return domain.BucketRetail, true, false
	}
	return domain.BucketHospitality, true, false
}

// lxpSignals detects operational / LXP buying signals per ICP doc §4–5 (training, compliance, scale).
func lxpSignals(ctx string) bool {
	return lxpSignalsForKeys(ctx, nil)
}

// lxpSignalsForKeys uses catalog signal groups; nil or empty keys uses the full ICP keyword union.
func lxpSignalsForKeys(ctx string, signalKeys []string) bool {
	groups := signalKeywordsByIDs(signalKeys)
	lower := strings.ToLower(ctx)
	for _, g := range groups {
		for _, kw := range g {
			if strings.Contains(lower, kw) {
				return true
			}
		}
	}
	return false
}

func profileDisqualifiers(ctx string) []string {
	return profileDisqualifiersSegments(ctx, nil)
}

// profileDisqualifiersSegments enforces only the selected catalog exclusion IDs; empty = skip segment-based DQs.
func profileDisqualifiersSegments(ctx string, segmentKeys []string) []string {
	if len(segmentKeys) == 0 {
		return nil
	}
	lower := strings.ToLower(ctx)
	want := map[string]struct{}{}
	for _, id := range segmentKeys {
		want[id] = struct{}{}
	}
	var out []string
	for _, ex := range CatalogExclusions() {
		if _, ok := want[ex.ID]; !ok {
			continue
		}
		for _, kw := range ex.Keywords {
			if strings.Contains(lower, kw) {
				out = append(out, "disqualifying business profile")
				return out
			}
		}
	}
	return out
}
