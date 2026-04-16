package extraction

import (
	"fmt"
	"regexp"
	"strings"
)

var tagLine = regexp.MustCompile(`(?i)^\s*@(company|industry|size|location)\s*:\s*(.+?)\s*$`)

// Size: conservative heuristic only.
var reOverEmployees = regexp.MustCompile(`(?i)\bover\s+([0-9]{3,})\s+employees\b|\bemployee\s+base\s+over\s+([0-9]{3,})\b|\bover\s+1k\b`)
var reTildeEmployees = regexp.MustCompile(`(?i)~\s*([0-9]{2,4})\s+employees`)
var reRangeEmployees = regexp.MustCompile(`(?i)est\.\s*([0-9]{2,4})\s*[–-]\s*([0-9]{2,4})`)

var (
	reLocationIndonesia = regexp.MustCompile(`(?i)\bindonesia\b`)
	reLocationJakarta   = regexp.MustCompile(`(?i)\bjakarta\b`)
	reLocationTier1     = regexp.MustCompile(`(?i)\btier-1\s+cities\b`)
)

// Company name fallbacks (only when @company is absent). Order: PT → Bank → Corp → Group.
var (
	reCompanyPT = regexp.MustCompile(`(?i)(PT\s+[A-Za-z][A-Za-z0-9&\-\.]*(?:\s+[A-Za-z][A-Za-z0-9&\-\.]*){0,6})`)
	reCompanyBank = regexp.MustCompile(`(?i)\b([A-Za-z0-9][A-Za-z0-9 &\-\.]{2,80})\s+Bank\b`)
	reCompanyCorp = regexp.MustCompile(`(?i)\b([A-Za-z0-9][A-Za-z0-9 &\-\.]{2,80})\s+Corp\b`)
	reCompanyGroup = regexp.MustCompile(`(?i)\b([A-Za-z0-9][A-Za-z0-9 &\-\.]{2,80})\s+Group\b`)
)

func extractTaggedFields(text string) (taggedFields, string) {
	out := taggedFields{}
	kept := make([]string, 0)

	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(line, "\r")
		m := tagLine.FindStringSubmatch(line)
		if len(m) != 3 {
			kept = append(kept, line)
			continue
		}

		key := strings.ToLower(strings.TrimSpace(m[1]))
		val := normalizeValue(m[2])
		if val == "" {
			continue
		}

		out.hasAnyTag = true
		switch key {
		case "company":
			out.companies = append(out.companies, val)
		case "industry":
			v := strings.ToLower(val)
			out.industry = &v
		case "size":
			v := val
			out.size = &v
		case "location":
			v := val
			out.location = &v
		}
	}

	return out, strings.Join(kept, "\n")
}

func extractHeuristicSignals(text string) heuristicSignals {
	clean := strings.TrimSpace(text)
	if clean == "" {
		return heuristicSignals{}
	}

	return heuristicSignals{
		industry: extractIndustrySignal(clean),
		size:     extractConservativeSize(clean),
		location: extractLocationSignal(clean),
	}
}

func extractIndustrySignal(text string) IndustrySignal {
	lower := strings.ToLower(text)

	banking := containsAny(lower, "bank", "banking", "compliance-heavy", "bumn")
	retail := containsAny(lower, "retail", "grocery", "supermarket", "department store", "outlet", "quick-service", "franchise")
	hospitality := containsAny(lower, "hospitality", "hotel", "housekeeping")

	count := 0
	if banking {
		count++
	}
	if retail {
		count++
	}
	if hospitality {
		count++
	}

	if count == 0 {
		return IndustrySignal{}
	}
	if count > 1 {
		return IndustrySignal{Value: nil, Confidence: "low", Ambiguous: true}
	}

	if banking {
		v := "banking"
		return IndustrySignal{Value: &v, Confidence: "low", Ambiguous: false}
	}
	if retail {
		v := "retail"
		return IndustrySignal{Value: &v, Confidence: "low", Ambiguous: false}
	}
	v := "hospitality"
	return IndustrySignal{Value: &v, Confidence: "low", Ambiguous: false}
}

func extractConservativeSize(text string) *string {
	if m := reTildeEmployees.FindStringSubmatch(text); len(m) == 2 {
		v := fmt.Sprintf("over %s employees", m[1])
		return &v
	}
	if m := reRangeEmployees.FindStringSubmatch(text); len(m) == 3 {
		v := fmt.Sprintf("%s–%s employees (estimated range)", m[1], m[2])
		return &v
	}
	m := reOverEmployees.FindStringSubmatch(text)
	if len(m) == 0 {
		return nil
	}

	if strings.Contains(strings.ToLower(m[0]), "over 1k") {
		v := "over 1000 employees"
		return &v
	}

	for _, g := range m[1:] {
		g = strings.TrimSpace(g)
		if g == "" {
			continue
		}
		v := fmt.Sprintf("over %s employees", g)
		return &v
	}
	return nil
}

func extractLocationSignal(text string) *string {
	lower := strings.ToLower(text)
	switch {
	case reLocationJakarta.MatchString(lower):
		v := "jakarta"
		return &v
	case reLocationIndonesia.MatchString(lower):
		v := "indonesia"
		return &v
	case reLocationTier1.MatchString(lower):
		v := "tier-1 cities"
		return &v
	default:
		return nil
	}
}

// extractCompanyNameFallback returns a name only when a strict pattern matches (deterministic).
func extractCompanyNameFallback(fullText string) *string {
	text := strings.TrimSpace(fullText)
	if text == "" {
		return nil
	}
	for _, re := range []*regexp.Regexp{reCompanyPT, reCompanyBank, reCompanyCorp, reCompanyGroup} {
		m := re.FindStringSubmatch(text)
		if len(m) < 2 {
			continue
		}
		name := normalizeValue(m[1])
		name = stripCompanyFallbackNoise(name)
		if name != "" {
			return &name
		}
	}
	return nil
}

func stripCompanyFallbackNoise(name string) string {
	name = strings.TrimSpace(name)
	lower := strings.ToLower(name)
	for _, cut := range []string{" operates nationwide", " operates", " estimated", " nationwide"} {
		if idx := strings.Index(lower, cut); idx > 0 {
			name = strings.TrimSpace(name[:idx])
			lower = strings.ToLower(name)
		}
	}
	return name
}

// extractHeadlineCompany uses the first clause of the first line (before ';') as a display name
// when no stricter pattern matched — keeps mock discovery rows identifiable.
// Requires a semicolon so generic one-line blurbs ("we help companies grow") are not used as names.
func extractHeadlineCompany(fullText string) *string {
	line := strings.TrimSpace(strings.Split(fullText, "\n")[0])
	if line == "" {
		return nil
	}
	parts := strings.SplitN(line, ";", 2)
	if len(parts) < 2 {
		return nil
	}
	seg := strings.TrimSpace(parts[0])
	if len(seg) < 10 || len(seg) > 72 {
		return nil
	}
	seg = normalizeValue(seg)
	if seg == "" {
		return nil
	}
	return &seg
}

func normalizeCompanyNames(values []string) (*string, string) {
	if len(values) == 0 {
		return nil, ""
	}

	norm := make([]string, 0, len(values))
	for _, v := range values {
		n := normalizeValue(v)
		if n != "" {
			norm = append(norm, n)
		}
	}
	if len(norm) == 0 {
		return nil, ""
	}

	first := norm[0]
	if len(norm) == 1 {
		return &first, ""
	}

	allSame := true
	for _, v := range norm[1:] {
		if !equalFoldTrim(first, v) {
			allSame = false
			break
		}
	}
	if allSame {
		return &first, ""
	}

	note := fmt.Sprintf("Multiple company names normalized to first: %s.", first)
	return &first, note
}

func normalizeValue(s string) string {
	return strings.TrimSpace(strings.Join(strings.Fields(s), " "))
}

func containsAny(text string, words ...string) bool {
	for _, w := range words {
		if strings.Contains(text, w) {
			return true
		}
	}
	return false
}

func equalFoldTrim(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}
