// Package exportcsv writes the leads CSV export (same columns as the legacy /export.csv).
package exportcsv

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"
	"time"

	"salesradar/internal/store"
)

// WriteResponse writes CSV rows for the given leads with Content-Disposition attachment.
func WriteResponse(w http.ResponseWriter, leads []store.Lead) {
	fn := fmt.Sprintf("leads_export_%s.csv", time.Now().UTC().Format("20060102"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fn))
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{
		"company", "industry", "size", "employee_size", "official_domain", "website_domain", "country_region", "reason_for_fit", "why_now", "why_now_strength", "sales_angle", "priority_score", "data_completeness",
		"action", "sales_ready", "sales_status", "icp_match", "duplicate_status", "lead_status", "confidence", "summary", "accept_explanation",
		"reasons", "missing_optional", "source", "source_ref", "created_at",
		"source_trace", "used_google", "used_apollo", "used_linkedin",
	})
	for _, l := range leads {
		reasons := strings.Join(l.Reasons, " | ")
		miss := strings.Join(l.MissingOptional, " | ")
		sr := "false"
		if l.SalesReady {
			sr = "true"
		}
		trace := strings.Join(l.SourceTrace, " | ")
		ug, ua, ul := "false", "false", "false"
		if l.UsedGoogle {
			ug = "true"
		}
		if l.UsedApollo {
			ua = "true"
		}
		if l.UsedLinkedIn {
			ul = "true"
		}
		_ = cw.Write([]string{
			l.Company,
			l.Industry,
			l.Size,
			l.EmployeeSize,
			l.OfficialDomain,
			l.WebsiteDomain,
			l.CountryRegion,
			l.ReasonForFit,
			l.WhyNow,
			l.WhyNowStrength,
			l.SalesAngle,
			fmt.Sprintf("%d", l.PriorityScore),
			fmt.Sprintf("%d", l.DataCompleteness),
			l.Action,
			sr,
			l.SalesStatus,
			l.ICPMatch,
			l.DuplicateStatus,
			l.LeadStatus,
			l.Confidence,
			l.Summary,
			l.AcceptExplanation,
			reasons,
			miss,
			l.Source,
			l.SourceRef,
			l.CreatedAt.UTC().Format(time.RFC3339),
			trace,
			ug, ua, ul,
		})
	}
	cw.Flush()
}
