package request

import (
	"net/http"
	"strings"

	"salesradar/internal/store"
)

// ParseListFilter parses list query parameters shared by HTML and JSON APIs.
func ParseListFilter(r *http.Request) store.ListFilter {
	q := r.URL.Query()
	f := store.ListFilter{
		Query:       strings.TrimSpace(q.Get("q")),
		ICPMatch:    q.Get("icp_match"),
		LeadStatus:  q.Get("lead_status"),
		SalesStatus: q.Get("sales_status"),
		Industry:    strings.TrimSpace(q.Get("industry")),
		Action:      strings.TrimSpace(q.Get("action")),
		SortBy:      q.Get("sort"),
		OrderAsc:    strings.ToLower(q.Get("order")) != "desc",
	}
	if f.SortBy != "priority" && f.SortBy != "confidence" && f.SortBy != "completeness" && f.SortBy != "action" && f.SortBy != "company" {
		f.SortBy = "priority"
	}
	return f
}
