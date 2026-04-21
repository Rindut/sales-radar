package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"salesradar/internal/apollo"
	"salesradar/internal/domain"
)

func discoverApollo(ctx context.Context, p domain.RunParams) ([]domain.RawCandidate, error) {
	apiKey := strings.TrimSpace(apollo.APIKeyFromEnv())
	if apiKey == "" {
		return nil, ErrApolloNotConfigured
	}
	limit := BatchLimit(p)
	filters := apollo.FiltersFromICP(p.ICPRuntime, limit)
	orgs, err := apollo.SearchCompanies(ctx, apiKey, filters)
	if err != nil {
		return nil, fmt.Errorf("discovery apollo: %w", err)
	}
	if len(orgs) == 0 {
		slog.Warn("discovery apollo: Apollo returned 0 results",
			"industries", strings.Join(filters.Industries, ","),
			"location", strings.TrimSpace(filters.Location),
			"keyword", strings.TrimSpace(filters.Keyword),
			"limit", filters.Limit,
		)
	}
	out := make([]domain.RawCandidate, 0, len(orgs))
	for i, org := range orgs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		name := strings.TrimSpace(org.Name)
		domainHost := strings.TrimSpace(org.PrimaryDomain)
		if name == "" && domainHost == "" {
			continue
		}
		sourceRef := strings.TrimSpace(org.ID)
		if sourceRef != "" {
			sourceRef = "apollo:organization:" + sourceRef
		}
		if sourceRef == "" && domainHost != "" {
			sourceRef = "https://" + domainHost + "/"
		}
		var lines []string
		if name != "" {
			lines = append(lines, fmt.Sprintf("@company: %s", name))
		}
		if domainHost != "" {
			lines = append(lines, fmt.Sprintf("@domain: %s", domainHost))
		}
		if strings.TrimSpace(org.Industry) != "" {
			lines = append(lines, fmt.Sprintf("@industry: %s", strings.TrimSpace(org.Industry)))
		}
		if org.EstimatedNumEmployees > 0 {
			lines = append(lines, fmt.Sprintf("@size: over %d employees", org.EstimatedNumEmployees))
		}
		if strings.TrimSpace(org.Location) != "" {
			lines = append(lines, fmt.Sprintf("@location: %s", strings.TrimSpace(org.Location)))
		}
		if strings.TrimSpace(org.ShortDescription) != "" {
			lines = append(lines, fmt.Sprintf("@snippet: %s", strings.TrimSpace(org.ShortDescription)))
		}
		if strings.TrimSpace(org.LinkedInURL) != "" {
			lines = append(lines, fmt.Sprintf("@linkedin_company_url: %s", strings.TrimSpace(org.LinkedInURL)))
		}
		lines = append(lines, "@data_completeness: high")

		out = append(out, domain.RawCandidate{
			DiscoveryID:         fmt.Sprintf("apollo-%d-%s", i+1, apolloDiscoveryID(name, domainHost, sourceRef)),
			Source:              domain.SourceApollo,
			SourceRef:           sourceRef,
			UnstructuredContext: strings.Join(lines, "\n"),
			OfficialDomain:      domainHost,
			EnrichedLinkedInURL: strings.TrimSpace(org.LinkedInURL),
			ProspectTrace: domain.ProspectTrace{
				UsedApollo: true,
				SourceTrace: []string{
					domain.TraceApolloDiscovery,
				},
			},
		})
	}
	return out, nil
}

func apolloDiscoveryID(name, domainHost, sourceRef string) string {
	base := strings.TrimSpace(strings.ToLower(domainHost))
	if base == "" {
		base = strings.TrimSpace(strings.ToLower(name))
	}
	if base == "" {
		base = strings.TrimSpace(strings.ToLower(sourceRef))
	}
	base = strings.ReplaceAll(base, " ", "-")
	base = strings.ReplaceAll(base, "/", "-")
	if base == "" {
		return "candidate"
	}
	return base
}
