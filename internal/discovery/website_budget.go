package discovery

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// WebsiteEnrichTotalBudget is the max wall time for the entire in-pool website pass (all domains).
// After this, remaining rows are left unenriched (best-effort).
func WebsiteEnrichTotalBudget() time.Duration {
	return durationSecEnv("SALESRADAR_WEBSITE_ENRICH_TOTAL_BUDGET_SEC", 600, 30, 3600)
}

// WebsiteEnrichPerCandidate caps Firecrawl + first-pass parse for one row before legacy HTTP.
func WebsiteEnrichPerCandidate() time.Duration {
	return durationSecEnv("SALESRADAR_WEBSITE_ENRICH_PER_CANDIDATE_SEC", 150, 20, 600)
}

func durationSecEnv(key string, def, min, max int) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return time.Duration(def) * time.Second
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < min {
		return time.Duration(def) * time.Second
	}
	if n > max {
		n = max
	}
	return time.Duration(n) * time.Second
}
