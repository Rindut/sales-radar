// Package enrichment is Module 8 — deterministic enrichment after extraction (no external I/O).
package enrichment

import (
	"context"
	"strings"
	"unicode/utf8"

	"salesradar/internal/domain"
	"salesradar/internal/enrich"
)

// Apply fills display-oriented fields on ExtractedLead when still empty (description, location hints).
func Apply(ctx context.Context, e *domain.ExtractedLead) {
	if e == nil {
		return
	}
	if err := ctx.Err(); err != nil {
		return
	}
	blob := strings.TrimSpace(e.UnstructuredContext)
	if e.AISummaryShort == nil && blob != "" {
		if s := summarizeFromUntaggedBody(blob); s != "" {
			e.AISummaryShort = &s
		}
	}
	locHint := blob
	if e.CompanyName != nil && strings.TrimSpace(*e.CompanyName) != "" {
		locHint += " " + strings.TrimSpace(*e.CompanyName)
	}
	if e.Location == nil || strings.TrimSpace(*e.Location) == "" {
		if r := enrich.CountryRegionFromText(locHint); r != "" {
			e.Location = &r
		}
	}
}

func summarizeFromUntaggedBody(ctx string) string {
	var parts []string
	for _, line := range strings.Split(ctx, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "@") {
			continue
		}
		parts = append(parts, line)
	}
	s := strings.Join(parts, " ")
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return ""
	}
	if utf8.RuneCountInString(s) <= 240 {
		return s
	}
	return trimRunes(s, 240) + "…"
}

func trimRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	var b strings.Builder
	n := 0
	for _, r := range s {
		if n >= max {
			break
		}
		b.WriteRune(r)
		n++
	}
	return b.String()
}
