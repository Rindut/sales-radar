package firecrawl

import (
	"strings"
)

// BuildSalesSummary produces a short sales-oriented summary (not raw page dump).
func BuildSalesSummary(combinedMarkdown string) string {
	s := strings.TrimSpace(combinedMarkdown)
	if s == "" {
		return ""
	}
	flat := strings.Join(strings.Fields(s), " ")
	if len(flat) > 420 {
		flat = flat[:420] + "…"
	}
	return flat
}

// BuildSalesSignals extracts concise intelligence tags from page text.
func BuildSalesSignals(combinedMarkdown string) string {
	low := strings.ToLower(combinedMarkdown)
	var tags []string
	add := func(cond bool, label string) {
		if cond {
			tags = append(tags, label)
		}
	}
	add(strings.Contains(low, "compliance") || strings.Contains(low, "regulated") || strings.Contains(low, "regulator"),
		"regulated/compliance context")
	add(strings.Contains(low, "distributed") || strings.Contains(low, "remote workforce") || strings.Contains(low, "hybrid work"),
		"distributed workforce")
	add(strings.Contains(low, "academy") || strings.Contains(low, "learning and development") || strings.Contains(low, "l&d") ||
		strings.Contains(low, "training program") || strings.Contains(low, "upskill"),
		"training/academy relevance")
	add(strings.Contains(low, "we are hiring") || strings.Contains(low, "open positions") || strings.Contains(low, "careers") ||
		strings.Contains(low, "vacancies") || strings.Contains(low, "join our team"),
		"hiring activity")
	add(strings.Contains(low, "enterprise") || strings.Contains(low, "fortune") || strings.Contains(low, "global offices") ||
		strings.Contains(low, "worldwide") || strings.Contains(low, "10,000+") || strings.Contains(low, "10000"),
		"enterprise scale hints")
	if len(tags) == 0 {
		return "general corporate web presence"
	}
	return strings.Join(tags, "; ")
}
