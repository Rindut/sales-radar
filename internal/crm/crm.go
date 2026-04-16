// Package crm maps review output into a CRM-shaped payload (e.g. future Odoo create).
// No network calls — structure only.
package crm

import (
	"strings"

	"salesradar/internal/review"
)

// CRMLead is the outbound shape for CRM integration (name aligned with typical lead fields).
type CRMLead struct {
	Name        string `json:"name"`
	Industry    string `json:"industry"`
	CompanySize string `json:"company_size"`
	LeadStatus  string `json:"lead_status"`
	Confidence  string `json:"confidence"`
	Action      string `json:"action"`
	Notes       string `json:"notes"`
}

// MapToCRMLead converts a ReviewLead into a CRM-ready record (summary + reasons in Notes).
func MapToCRMLead(r review.ReviewLead) CRMLead {
	return CRMLead{
		Name:        safeString(r.Company),
		Industry:    safeString(r.Industry),
		CompanySize: r.Size,
		LeadStatus:  r.LeadStatus,
		Confidence:  r.Confidence,
		Action:      r.Action,
		Notes:       buildNotes(r),
	}
}

func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func buildNotes(r review.ReviewLead) string {
	var parts []string
	if t := strings.TrimSpace(r.Summary); t != "" {
		parts = append(parts, t)
	}
	if len(r.Reasons) > 0 {
		parts = append(parts, strings.Join(r.Reasons, " | "))
	}
	return strings.Join(parts, "\n")
}
