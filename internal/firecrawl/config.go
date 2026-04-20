package firecrawl

import (
	"os"
	"strconv"
	"strings"
)

const (
	apiBaseURL = "https://api.firecrawl.dev/v1"
	// EnvMaxPages caps pages scraped per company (MVP guardrail).
	EnvMaxPages = "SALESRADAR_FIRECRAWL_MAX_PAGES"
)

// APIKeyFromEnv reads FIRECRAWL_API_KEY.
func APIKeyFromEnv() string {
	return strings.TrimSpace(os.Getenv("FIRECRAWL_API_KEY"))
}

// Configured is true when Firecrawl API key is present.
func Configured() bool {
	return APIKeyFromEnv() != ""
}

// MaxPagesPerCompany returns max scrape URLs per company (default 5, max 15).
func MaxPagesPerCompany() int {
	const def = 5
	const hardMax = 15
	raw := strings.TrimSpace(os.Getenv(EnvMaxPages))
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return def
	}
	if n > hardMax {
		return hardMax
	}
	return n
}
