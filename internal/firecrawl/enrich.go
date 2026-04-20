package firecrawl

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"salesradar/internal/domain"
)

// EnrichWebsite uses Firecrawl map + selective scrape (no recursive crawl).
// On any error it returns a non-nil WebsiteEnrichment with Status=failed and empty text fields;
// callers may still apply legacy HTTP fetch.
func EnrichWebsite(ctx context.Context, host string) (*domain.WebsiteEnrichment, string) {
	now := time.Now().UTC().Format(time.RFC3339)
	out := &domain.WebsiteEnrichment{
		Status:     "failed",
		ReasonCode: "unknown_provider_error",
		EnrichedAt: now,
	}
	op := EnrichOperationTimeout()
	ctx, cancel := context.WithTimeout(ctx, op)
	defer cancel()

	if !Configured() {
		out.Status = "skipped"
		out.ReasonCode = "provider_not_configured"
		out.ReasonMessage = "Firecrawl API key is not configured."
		return out, ""
	}
	host = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(host, "https://"), "http://"))
	if host == "" {
		out.Status = "skipped"
		out.ReasonCode = "no_valid_domain"
		out.ReasonMessage = "No valid host/domain provided."
		return out, ""
	}

	t0 := time.Now()
	slog.Info("firecrawl: enrich start", "host", host, "op_timeout_sec", int(op.Seconds()))
	defer func() {
		slog.Info("firecrawl: enrich end", "host", host, "elapsed_ms", time.Since(t0).Milliseconds(), "status", out.Status)
	}()

	root := "https://" + host + "/"

	max := MaxPagesPerCompany()

	mapBody, status, err := postJSON(ctx, "/map", map[string]any{
		"url":               root,
		"limit":             120,
		"ignoreSitemap":     false,
		"includeSubdomains": false,
	})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			out.Status = "failed"
			out.ReasonCode = "provider_timeout"
			out.ReasonMessage = "Firecrawl map request exceeded timeout."
			out.Summary = "Firecrawl map: deadline exceeded"
		} else {
			out.ReasonCode = "unknown_provider_error"
			out.ReasonMessage = "Firecrawl map request failed."
			out.ErrorMessage = err.Error()
			out.Summary = "Firecrawl map: " + err.Error()
		}
		return out, ""
	}
	if status < 200 || status >= 300 {
		if status == 403 {
			out.ReasonCode = "provider_forbidden"
			out.ReasonMessage = "Firecrawl rejected request with forbidden status."
		} else {
			out.ReasonCode = "provider_http_error"
			out.ReasonMessage = "Firecrawl map returned non-2xx status."
		}
		out.ErrorMessage = fmt.Sprintf("map_http_status=%d", status)
		out.Summary = fmt.Sprintf("Firecrawl map HTTP %d", status)
		return out, ""
	}
	links, err := parseMapLinks(mapBody)
	if err != nil {
		out.ReasonCode = "unknown_provider_error"
		out.ReasonMessage = "Failed to parse Firecrawl map response."
		out.ErrorMessage = err.Error()
		out.Summary = err.Error()
		return out, ""
	}

	selected := pickURLs(root, links, max)
	if len(selected) == 0 {
		out.Status = "failed"
		out.ReasonCode = "no_public_text"
		out.ReasonMessage = "No suitable public pages were selected."
		out.Summary = "No suitable public pages selected for scraping."
		return out, ""
	}
	out.SelectedURLs = append([]string(nil), selected...)
	out.PagesAttempted = len(selected)

	var parts []string
	var scrapeOK int
	for _, pageURL := range selected {
		if err := ctx.Err(); err != nil {
			slog.Info("firecrawl: stopping scrapes (deadline or cancel)", "host", host, "err", err)
			break
		}
		scBody, st, err := postJSON(ctx, "/scrape", map[string]any{
			"url":             pageURL,
			"formats":         []string{"markdown"},
			"onlyMainContent": true,
		})
		if err != nil {
			slog.Warn("firecrawl: scrape request error", "host", host, "url", pageURL, "err", err)
			if errors.Is(err, context.DeadlineExceeded) {
				break
			}
			continue
		}
		if st < 200 || st >= 300 {
			continue
		}
		md, err := parseScrapeMarkdown(scBody)
		if err != nil || md == "" {
			continue
		}
		scrapeOK++
		parts = append(parts, "— "+pageURL+" —\n"+md)
	}
	combined := strings.TrimSpace(strings.Join(parts, "\n\n"))
	if combined == "" {
		out.Status = "failed"
		if ctx.Err() != nil {
			out.ReasonCode = "provider_timeout"
			out.ReasonMessage = "Firecrawl timed out before scrape completed."
			out.ErrorMessage = ctx.Err().Error()
			out.Summary = "Firecrawl: timed out before usable content (" + ctx.Err().Error() + ")"
		} else {
			out.ReasonCode = "provider_http_error"
			out.ReasonMessage = "Firecrawl returned no extractable markdown."
			out.Summary = "Firecrawl returned no extractable markdown for selected pages."
		}
		return out, ""
	}

	out.Summary = BuildSalesSummary(combined)
	out.Signals = BuildSalesSignals(combined)
	out.PagesSucceeded = scrapeOK
	out.Status = "success"
	out.ReasonCode = ""
	out.ReasonMessage = ""
	out.ErrorMessage = ""
	return out, combined
}
