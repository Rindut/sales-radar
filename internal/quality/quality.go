// Package quality gates which leads are persisted (deterministic rules).
package quality

import (
	"strings"

	"salesradar/internal/companycheck"
	"salesradar/internal/review"
)

// RequiredMissing lists blocking gaps for a stored lead (MVP: company + industry + one enrich signal).
func RequiredMissing(rl review.ReviewLead) []string {
	var m []string
	if rl.Company == nil || strings.TrimSpace(*rl.Company) == "" {
		m = append(m, "company_name")
	}
	if strings.TrimSpace(rl.OfficialDomain) == "" {
		m = append(m, "official_domain")
	}
	if rl.Industry == nil || strings.TrimSpace(*rl.Industry) == "" {
		m = append(m, "industry")
	}
	if strings.TrimSpace(rl.ReasonForFit) == "" {
		m = append(m, "reason_for_fit")
	}
	if !hasAtLeastOneEnrichment(rl) {
		m = append(m, "employee_size_or_website_or_reason_for_fit")
	}
	return m
}

func hasAtLeastOneEnrichment(rl review.ReviewLead) bool {
	hasSize := rl.EmployeeSize != "" && rl.EmployeeSize != "—"
	hasWeb := strings.TrimSpace(rl.OfficialDomain) != "" || strings.TrimSpace(rl.WebsiteDomain) != ""
	hasReason := strings.TrimSpace(rl.ReasonForFit) != ""
	// "unknown" counts as an explicit employee_size value for MVP (field present, enrichment partial).
	return hasSize || hasWeb || hasReason
}

// PassesStorageGate: real identifiable company + industry + at least one of (size incl. unknown, website, reason_for_fit).
// Discards generic labels, abstract categories, and names that do not look like real organizations.
func PassesStorageGate(rl review.ReviewLead) bool {
	if len(RequiredMissing(rl)) != 0 {
		return false
	}
	if rl.Company == nil || !companycheck.IsIdentifiableCompany(*rl.Company) {
		return false
	}
	if companycheck.IsBlockedNonCompanyDomain(strings.TrimSpace(rl.OfficialDomain)) {
		return false
	}
	return true
}
