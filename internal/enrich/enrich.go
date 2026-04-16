// Package enrich derives display fields from staged leads and source refs (deterministic, no external calls).
package enrich

import (
	"net/url"
	"regexp"
	"strings"

	"salesradar/internal/companycheck"
	"salesradar/internal/domain"
)

// WebsiteDomainFromRef returns the hostname without www prefix, or empty if not parseable as HTTP(S).
func WebsiteDomainFromRef(ref string) string {
	if ref == "" {
		return ""
	}
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		u, err := url.Parse(ref)
		if err != nil || u.Host == "" {
			return ""
		}
		return companycheck.SanitizeCompanyWebsiteDomain(u.Hostname())
	}
	// Non-HTTP refs: not a company-owned domain (Apollo org id is not a corporate website).
	if strings.HasPrefix(ref, "apollo:organization:") {
		return ""
	}
	if strings.HasPrefix(ref, "unknown-source-ref:") {
		return ""
	}
	return ""
}

// LinkedInCompanyURL returns a normalized company URL if ref is LinkedIn, else empty.
func LinkedInCompanyURL(ref string) string {
	if !strings.Contains(strings.ToLower(ref), "linkedin.com/company") {
		return ""
	}
	u, err := url.Parse(ref)
	if err != nil || u.Host == "" {
		return ""
	}
	u.Scheme = "https"
	u.Path = strings.TrimSuffix(u.Path, "/")
	return u.String()
}

var countryHints = []struct {
	re  *regexp.Regexp
	tag string
}{
	{regexp.MustCompile(`(?i)\bindonesia\b`), "Indonesia"},
	{regexp.MustCompile(`(?i)\bsingapore\b`), "Singapore"},
	{regexp.MustCompile(`(?i)\bmalaysia\b`), "Malaysia"},
	{regexp.MustCompile(`(?i)\bunited states\b|\busa\b|\bU\.S\.`), "United States"},
}

// CountryRegionFromText extracts a best-effort country/region from unstructured text.
func CountryRegionFromText(text string) string {
	for _, h := range countryHints {
		if h.re.MatchString(text) {
			return h.tag
		}
	}
	return ""
}

var lxpHintRE = regexp.MustCompile(`(?i)training|onboarding|compliance|learning|l&d|frontline|workforce|branch|outlet|turnover|certification`)

// ReasonForFit builds a short sales-ready line (LXP / training angle) from ICP + discovery text.
func ReasonForFit(icp domain.ICPMatch, industry string, reasons []string, contextBlob string) string {
	var parts []string
	if industry != "" && industry != "—" {
		parts = append(parts, asciiTitle(industry)+" account")
	}
	switch icp {
	case domain.ICPYes:
		parts = append(parts, "strong ICP alignment for LXP rollout")
	case domain.ICPPartial:
		parts = append(parts, "medium ICP alignment with clear enablement potential")
	case domain.ICPNo:
		parts = append(parts, "outside core ICP")
	}
	ctx := strings.ToLower(strings.TrimSpace(contextBlob))
	if strings.Contains(ctx, "job_signal_detected") {
		parts = append(parts, "Active hiring indicates training need")
	}
	if strings.Contains(ctx, "growth_signal_detected") {
		parts = append(parts, "Growth phase indicates scaling needs")
	}
	switch {
	case strings.Contains(ctx, "multi-branch") || strings.Contains(ctx, "branch network") || strings.Contains(ctx, "branch"):
		parts = append(parts, "branch-based operations likely need standardized frontline training and certification")
	case strings.Contains(ctx, "outlet") || strings.Contains(ctx, "store"):
		parts = append(parts, "multi-outlet execution suggests a need for consistent onboarding and role-based learning paths")
	case strings.Contains(ctx, "compliance"):
		parts = append(parts, "compliance workload is a strong indicator for trackable learning and audit-ready completion records")
	case strings.Contains(ctx, "turnover") || (strings.Contains(ctx, "hiring") && !strings.Contains(ctx, "job_signal_detected")):
		parts = append(parts, "active hiring/turnover indicates recurring onboarding demand suited for an LXP")
	case strings.Contains(ctx, "hospitality") || strings.Contains(ctx, "hotel"):
		parts = append(parts, "service-quality consistency across properties can benefit from repeatable digital training journeys")
	case strings.Contains(ctx, "bank"):
		parts = append(parts, "distributed banking workforce often requires recurring policy and product knowledge reinforcement")
	case strings.Contains(ctx, "retail") || strings.Contains(ctx, "fmcg") || strings.Contains(ctx, "grocery"):
		parts = append(parts, "retail operations with frontline teams typically benefit from continuous microlearning and SOP refreshers")
	}
	for _, r := range reasons {
		if len(r) > 0 && len(strings.Join(parts, "; ")) < 180 {
			parts = append(parts, r)
		}
	}
	if lxpHintRE.MatchString(contextBlob) {
		parts = append(parts, "operational context suggests measurable impact from structured LXP programs")
	}
	s := strings.Join(parts, "; ")
	s = strings.TrimSpace(s)
	if s == "" {
		return "Target-sector lead; confirm training/LXP need with stakeholder"
	}
	if len(s) > 180 {
		return s[:177] + "..."
	}
	return s
}

// DefaultReasonForFit ensures a non-empty line when ICP/context produced nothing (MVP safety net).
func DefaultReasonForFit(icp domain.ICPMatch, industry string) string {
	s := ReasonForFit(icp, industry, nil, "")
	if strings.TrimSpace(s) == "" {
		return "ICP-assessed lead; validate LXP/training fit with discovery notes"
	}
	return s
}

func asciiTitle(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if len(s) == 1 {
		return strings.ToUpper(s)
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
}

func clampInt(n, lo, hi int) int {
	if n < lo {
		return lo
	}
	if n > hi {
		return hi
	}
	return n
}

// DataCompleteness returns 0–100 from available signals.
func DataCompleteness(company, employeeSize, websiteDomain, country, industry, linkedInURL string) int {
	score := 0
	if company != "" && company != "—" {
		score += 25
	}
	switch {
	case employeeSize != "" && employeeSize != "—" && employeeSize != "unknown":
		score += 20
	case employeeSize == "unknown":
		score += 8
	}
	if websiteDomain != "" {
		score += 20
	}
	if country != "" {
		score += 15
	}
	if industry != "" && industry != "—" {
		score += 10
	}
	if linkedInURL != "" {
		score += 10
	}
	return clampInt(score, 0, 100)
}
