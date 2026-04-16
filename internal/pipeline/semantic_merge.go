package pipeline

import (
	"strings"

	"salesradar/internal/companycheck"
	"salesradar/internal/domain"
)

func rowRank(p PreparedRow) int {
	n := 0
	if p.Staged.ICPMatch == domain.ICPYes {
		n += 1000
	} else if p.Staged.ICPMatch == domain.ICPPartial {
		n += 100
	}
	n += p.Review.DataCompleteness
	if p.Review.OfficialDomain != "" || p.Review.WebsiteDomain != "" {
		n += 50
	}
	if p.Review.SalesReady {
		n += 200
	}
	return n
}

// mergeSemanticRows keeps one row per company concept: same employer domain, or same normalized name
// (order-independent tokens so repeated phrases like "Boutique hotel collection" collapse to one).
func mergeSemanticRows(rows []PreparedRow) ([]PreparedRow, int) {
	if len(rows) <= 1 {
		return rows, 0
	}
	best := make(map[string]PreparedRow)
	for _, row := range rows {
		key := mergeBucketKey(row)
		prev, ok := best[key]
		if !ok || rowRank(row) > rowRank(prev) {
			best[key] = row
		}
	}
	merged := len(rows) - len(best)
	out := make([]PreparedRow, 0, len(best))
	for _, v := range best {
		out = append(out, v)
	}
	return out, merged
}

func mergeBucketKey(row PreparedRow) string {
	wd := strings.TrimSpace(row.Review.OfficialDomain)
	if wd == "" {
		wd = strings.TrimSpace(row.Review.WebsiteDomain)
	}
	if wd != "" {
		return "site:" + companycheck.NormalizeHost(wd)
	}
	comp := ""
	if row.Review.Company != nil {
		comp = *row.Review.Company
	}
	key := companycheck.MergeDedupKey(comp)
	if key != "" {
		return "name:" + key
	}
	return "id:" + row.Staged.DiscoveryID
}
