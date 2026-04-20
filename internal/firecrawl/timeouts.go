package firecrawl

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// HTTPOneShotTimeout bounds each Firecrawl map/scrape HTTP call (default 30s).
func HTTPOneShotTimeout() time.Duration {
	return durationFromEnv("SALESRADAR_FIRECRAWL_HTTP_TIMEOUT_SEC", 30, 5, 120)
}

// EnrichOperationTimeout bounds map + all scrapes for one host (default 2m).
func EnrichOperationTimeout() time.Duration {
	return durationFromEnv("SALESRADAR_FIRECRAWL_ENRICH_TIMEOUT_SEC", 120, 15, 600)
}

func durationFromEnv(key string, defSec, minSec, maxSec int) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return time.Duration(defSec) * time.Second
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < minSec {
		return time.Duration(defSec) * time.Second
	}
	if n > maxSec {
		n = maxSec
	}
	return time.Duration(n) * time.Second
}
