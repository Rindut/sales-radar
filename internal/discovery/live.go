package discovery

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"salesradar/internal/apollo"
	"salesradar/internal/companycheck"
	"salesradar/internal/domain"
	"salesradar/internal/googlesearch"
)

func discoverLive(ctx context.Context, cfg googlesearch.Config, p domain.RunParams, toggles domain.DiscoverySourceToggles) ([]domain.RawCandidate, error) {
	apolloKey := apollo.APIKeyFromEnv()
	limit := BatchLimit(p)
	seen := make(map[string]struct{})
	var out []domain.RawCandidate
	seq := 0

outer:
	for _, q := range icpSearchQueries() {
		if len(out) >= limit {
			break
		}
		results, err := cfg.Search(ctx, q, 20)
		if err != nil {
			return nil, fmt.Errorf("discovery live: %w", err)
		}
		for _, r := range results {
			if len(out) >= limit {
				break outer
			}
			host := domainFromResultURL(r.Link)
			host = companycheck.SanitizeCompanyWebsiteDomain(host)
			if host == "" {
				continue
			}
			if _, dup := seen[host]; dup {
				continue
			}
			seen[host] = struct{}{}

			companyName := companyNameFromTitle(r.Title)
			trace := domain.ProspectTrace{
				UsedGoogle:  true,
				SourceTrace: []string{"google_discovery"},
			}
			var enrichLi string
			if toggles.Apollo && apolloKey != "" {
				if org, err := apollo.EnrichByDomain(ctx, apolloKey, host); err == nil && org != nil {
					trace.UsedApollo = true
					trace.SourceTrace = append(trace.SourceTrace, "apollo_enrichment")
					if strings.TrimSpace(org.Name) != "" {
						companyName = strings.TrimSpace(org.Name)
					}
					if u := strings.TrimSpace(org.LinkedInURL); u != "" && toggles.LinkedIn {
						enrichLi = u
						trace.UsedLinkedIn = true
						trace.SourceTrace = append(trace.SourceTrace, "linkedin_validation")
					}
					if org.EstimatedNumEmployees > 0 {
						r.Snippet += fmt.Sprintf(" Estimated employees (Apollo): %d.", org.EstimatedNumEmployees)
					}
				}
			}

			industryHint := industryHintFromQuery(q)
			var lines []string
			lines = append(lines, fmt.Sprintf("@company: %s", companyName))
			lines = append(lines, fmt.Sprintf("@domain: %s", host))
			if industryHint != "" {
				lines = append(lines, fmt.Sprintf("@industry: %s", industryHint))
			}
			lines = append(lines, fmt.Sprintf("@snippet: %s", strings.TrimSpace(r.Snippet)))
			ctxBody := strings.Join(lines, "\n")

			seq++
			base := domain.RawCandidate{
				DiscoveryID:         fmt.Sprintf("live-%d-%s", seq, host),
				Source:              domain.SourceGoogle,
				SourceRef:           "https://" + host + "/",
				UnstructuredContext: ctxBody,
				OfficialDomain:      host,
				ProspectTrace:       trace,
				EnrichedLinkedInURL: enrichLi,
			}
			crawled := enrichWithWebsiteCrawl(ctx, base)
			for _, cand := range crawled {
				if len(out) >= limit {
					break outer
				}
				out = append(out, cand)
			}
		}
	}
	return out, nil
}

func domainFromResultURL(link string) string {
	u, err := url.Parse(link)
	if err != nil || u.Hostname() == "" {
		return ""
	}
	return u.Hostname()
}

func companyNameFromTitle(title string) string {
	t := strings.TrimSpace(title)
	for _, sep := range []string{" | ", " - ", " — ", " – "} {
		if i := strings.Index(t, sep); i > 0 {
			t = strings.TrimSpace(t[:i])
			break
		}
	}
	return t
}

func industryHintFromQuery(q string) string {
	ql := strings.ToLower(q)
	switch {
	case strings.Contains(ql, "bank"):
		return "banking"
	case strings.Contains(ql, "hotel") || strings.Contains(ql, "hospitality"):
		return "hospitality"
	case strings.Contains(ql, "retail"):
		return "retail"
	default:
		return ""
	}
}
