package discovery

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"salesradar/internal/domain"
	"salesradar/internal/failpoint"
	"salesradar/internal/firecrawl"
)

var reHTMLTag = regexp.MustCompile(`(?s)<[^>]*>`)
var reSpace = regexp.MustCompile(`\s+`)
var reBrandsLine = regexp.MustCompile(`(?i)(our brands|brands include|subsidiaries|business units?)[:\s]+([a-z0-9,\-&\s]{8,120})`)

func enrichWithWebsiteCrawl(ctx context.Context, in domain.RawCandidate, fpMode failpoint.WebsiteCrawlMode) []domain.RawCandidate {
	domainHost := strings.TrimSpace(in.OfficialDomain)
	if domainHost == "" {
		return []domain.RawCandidate{in}
	}
	if fpMode != failpoint.WebsiteCrawlNone {
		return enrichWithWebsiteCrawlFailpoint(in, domainHost, fpMode)
	}

	var we *domain.WebsiteEnrichment
	blob := ""

	if firecrawl.Configured() {
		slog.Info("website crawl: firecrawl starting", "host", domainHost)
		we, blob = firecrawl.EnrichWebsite(ctx, domainHost)
		slog.Info("website crawl: firecrawl finished", "host", domainHost, "has_blob", strings.TrimSpace(blob) != "")
	}

	// Legacy HTTP fetch when Firecrawl is off or returned no usable text (never fails the pipeline).
	// Uses an independent short deadline so a cancelled Firecrawl context does not skip legacy fetch.
	if strings.TrimSpace(blob) == "" {
		legacyCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		homeText := fetchPageText(legacyCtx, "https://"+domainHost+"/")
		aboutText := fetchPageText(legacyCtx, "https://"+domainHost+"/about")
		lb := strings.TrimSpace(homeText + " " + aboutText)
		if lb != "" {
			blob = lb
			now := time.Now().UTC().Format(time.RFC3339)
			if we == nil || we.Status != "success" {
				urls := []string{"https://" + domainHost + "/"}
				pagesSucceeded := 0
				if strings.TrimSpace(homeText) != "" {
					pagesSucceeded++
				}
				if strings.TrimSpace(aboutText) != "" {
					urls = append(urls, "https://"+domainHost+"/about")
					pagesSucceeded++
				}
				we = &domain.WebsiteEnrichment{
					SelectedURLs:   urls,
					Summary:        firecrawl.BuildSalesSummary(lb),
					Signals:        firecrawl.BuildSalesSignals(lb),
					PagesAttempted: len(urls),
					PagesSucceeded: pagesSucceeded,
					Status:         "legacy_fallback",
					ReasonCode:     "legacy_fallback",
					ReasonMessage:  "Used legacy HTTP homepage/about fallback.",
					EnrichedAt:     now,
				}
			}
		} else if we == nil {
			we = &domain.WebsiteEnrichment{
				PagesAttempted: 2,
				PagesSucceeded: 0,
				Status:         "skipped",
				ReasonCode:     "no_public_text",
				ReasonMessage:  "No public homepage/about text retrieved.",
				Summary:        "No public homepage text retrieved.",
				EnrichedAt:     time.Now().UTC().Format(time.RFC3339),
			}
		}
	}

	if strings.TrimSpace(blob) == "" {
		if we != nil {
			in.WebsiteEnrichment = we
		}
		return []domain.RawCandidate{in}
	}

	industry := inferIndustryFromWebsite(blob)
	sizeHint := inferEmployeeSizeHint(blob)
	kw := extractWebsiteKeywords(blob)
	product := inferProductKeywords(blob)

	lines := []string{
		in.UnstructuredContext,
		fmt.Sprintf("@website_description: %s", trimWords(blob, 40)),
	}
	if industry != "" {
		lines = append(lines, fmt.Sprintf("@industry: %s", industry))
	}
	if sizeHint != "" {
		lines = append(lines, fmt.Sprintf("@size: %s", sizeHint))
	}
	if kw != "" {
		lines = append(lines, fmt.Sprintf("@keywords: %s", kw))
	}
	if product != "" {
		lines = append(lines, fmt.Sprintf("@product_keywords: %s", product))
	}
	if we != nil && strings.TrimSpace(we.Summary) != "" {
		lines = append(lines, fmt.Sprintf("@website_enrichment_summary: %s", strings.TrimSpace(we.Summary)))
	}
	if we != nil && strings.TrimSpace(we.Signals) != "" {
		lines = append(lines, fmt.Sprintf("@website_enrichment_signals: %s", strings.TrimSpace(we.Signals)))
	}
	in.UnstructuredContext = strings.Join(lines, "\n")
	if !containsTrace(in.ProspectTrace.SourceTrace, "website_crawl") {
		in.ProspectTrace.SourceTrace = append(in.ProspectTrace.SourceTrace, "website_crawl")
	}
	in.WebsiteEnrichment = we

	out := []domain.RawCandidate{in}
	for i, brand := range detectBrands(blob) {
		if i >= 2 {
			break
		}
		b := in
		b.DiscoveryID = fmt.Sprintf("%s-brand-%d", in.DiscoveryID, i+1)
		b.Source = domain.SourceCompanyWebsite
		b.UnstructuredContext = strings.Join([]string{
			in.UnstructuredContext,
			fmt.Sprintf("@company: %s", brand),
			"@website_signal: brand/subsidiary detected",
		}, "\n")
		b.ProspectTrace.SourceTrace = append([]string{"website_crawl_discovery"}, in.ProspectTrace.SourceTrace...)
		b.WebsiteEnrichment = in.WebsiteEnrichment
		out = append(out, b)
	}
	return out
}

func enrichWithWebsiteCrawlFailpoint(in domain.RawCandidate, domainHost string, mode failpoint.WebsiteCrawlMode) []domain.RawCandidate {
	now := time.Now().UTC().Format(time.RFC3339)
	switch mode {
	case failpoint.WebsiteCrawlSuccess:
		urls := []string{
			"https://" + domainHost + "/",
			"https://" + domainHost + "/about",
		}
		blob := "company overview growth hiring operations and multi-branch expansion."
		in.WebsiteEnrichment = &domain.WebsiteEnrichment{
			SelectedURLs:   urls,
			Summary:        "Failpoint success: deterministic website summary.",
			Signals:        "Failpoint signal: expansion, hiring.",
			PagesAttempted: 2,
			PagesSucceeded: 2,
			Status:         "success",
			ReasonCode:     "failpoint_website_success",
			ReasonMessage:  "Website crawl success forced by verification failpoint.",
			EnrichedAt:     now,
		}
		in.UnstructuredContext = strings.TrimSpace(in.UnstructuredContext + "\n@website_description: " + blob)
		if !containsTrace(in.ProspectTrace.SourceTrace, "website_crawl") {
			in.ProspectTrace.SourceTrace = append(in.ProspectTrace.SourceTrace, "website_crawl")
		}
		return []domain.RawCandidate{in}
	case failpoint.WebsiteCrawlTimeout:
		in.WebsiteEnrichment = &domain.WebsiteEnrichment{
			PagesAttempted: 2,
			PagesSucceeded: 0,
			Status:         "failed",
			ReasonCode:     "provider_timeout",
			ReasonMessage:  "Website crawl timeout forced by verification failpoint.",
			ErrorMessage:   context.DeadlineExceeded.Error(),
			Summary:        "Failpoint timeout while crawling website.",
			EnrichedAt:     now,
		}
		return []domain.RawCandidate{in}
	case failpoint.WebsiteCrawlError:
		in.WebsiteEnrichment = &domain.WebsiteEnrichment{
			PagesAttempted: 1,
			PagesSucceeded: 0,
			Status:         "failed",
			ReasonCode:     "unknown_provider_error",
			ReasonMessage:  "Website crawl error forced by verification failpoint.",
			ErrorMessage:   errors.New("forced website crawl error").Error(),
			Summary:        "Failpoint provider error while crawling website.",
			EnrichedAt:     now,
		}
		return []domain.RawCandidate{in}
	default:
		return []domain.RawCandidate{in}
	}
}

func fetchPageText(ctx context.Context, u string) string {
	cctx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, http.MethodGet, u, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "SalesRadarBot/1.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ""
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 200000))
	if err != nil {
		return ""
	}
	t := strings.ToLower(string(body))
	t = reHTMLTag.ReplaceAllString(t, " ")
	t = reSpace.ReplaceAllString(t, " ")
	return strings.TrimSpace(t)
}

func inferIndustryFromWebsite(text string) string {
	switch {
	case hasAny(text, "bank", "banking", "financial services", "digital banking"):
		return "banking"
	case hasAny(text, "retail", "grocery", "supermarket", "department store", "fmcg"):
		return "retail"
	case hasAny(text, "hotel", "hospitality", "resort", "accommodation"):
		return "hospitality"
	default:
		return ""
	}
}

func inferEmployeeSizeHint(text string) string {
	switch {
	case hasAny(text, "10,000", "10000", "over 10000"):
		return "10000+"
	case hasAny(text, "5,000", "5000", "over 5000"):
		return "5000+"
	case hasAny(text, "1,000", "1000", "over 1000"):
		return "1000+"
	default:
		return ""
	}
}

func extractWebsiteKeywords(text string) string {
	var out []string
	for _, k := range []string{"expanding", "new branch", "outlet", "distributed workforce", "compliance", "hiring", "onboarding", "training"} {
		if strings.Contains(text, k) {
			out = append(out, k)
		}
	}
	return strings.Join(out, ", ")
}

func inferProductKeywords(text string) string {
	var out []string
	for _, k := range []string{"payments", "lending", "deposits", "omnichannel", "e-commerce", "guest services", "property management"} {
		if strings.Contains(text, k) {
			out = append(out, k)
		}
	}
	return strings.Join(out, ", ")
}

func detectBrands(text string) []string {
	m := reBrandsLine.FindStringSubmatch(text)
	if len(m) < 3 {
		return nil
	}
	chunks := strings.Split(m[2], ",")
	var out []string
	for _, c := range chunks {
		s := strings.TrimSpace(c)
		if len(s) < 3 {
			continue
		}
		out = append(out, titleCase(s))
	}
	return out
}

func trimWords(s string, n int) string {
	fs := strings.Fields(s)
	if len(fs) <= n {
		return strings.Join(fs, " ")
	}
	return strings.Join(fs[:n], " ")
}

func containsTrace(tr []string, v string) bool {
	for _, s := range tr {
		if s == v {
			return true
		}
	}
	return false
}

func titleCase(s string) string {
	parts := strings.Fields(strings.TrimSpace(s))
	for i := range parts {
		p := parts[i]
		if p == "" {
			continue
		}
		if len(p) == 1 {
			parts[i] = strings.ToUpper(p)
		} else {
			parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
		}
	}
	return strings.Join(parts, " ")
}

func hasAny(s string, words ...string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}
