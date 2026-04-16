package discovery

import (
	"fmt"
	"net/url"
	"strings"

	"salesradar/internal/companycheck"
	"salesradar/internal/domain"
)

type directoryEntry struct {
	CompanyName string
	Category    string
	Website     string
}

// curatedDirectory is a lightweight, deterministic directory-like source for structured company discovery.
var curatedDirectory = []directoryEntry{
	{CompanyName: "Matahari Department Store", Category: "retail", Website: "https://www.matahari.com/"},
	{CompanyName: "Alfamart", Category: "retail", Website: "https://alfamart.co.id/"},
	{CompanyName: "Indomaret", Category: "retail", Website: "https://www.indomaret.co.id/"},
	{CompanyName: "Bank Central Asia", Category: "banking", Website: "https://www.bca.co.id/"},
	{CompanyName: "Bank Mandiri", Category: "banking", Website: "https://bankmandiri.co.id/"},
	{CompanyName: "Bank Rakyat Indonesia", Category: "banking", Website: "https://bri.co.id/"},
	{CompanyName: "Archipelago International", Category: "hospitality", Website: "https://www.archipelagointernational.com/"},
	{CompanyName: "The Ascott Limited", Category: "hospitality", Website: "https://www.discoverasr.com/"},
}

// applyDirectoryDiscovery appends normalized directory-sourced candidates up to batch limit.
func applyDirectoryDiscovery(base []domain.RawCandidate, p domain.RunParams) []domain.RawCandidate {
	limit := BatchLimit(p)
	if len(base) >= limit {
		return base
	}
	out := make([]domain.RawCandidate, 0, limit)
	out = append(out, base...)

	seenDomain := map[string]struct{}{}
	seenCompany := map[string]struct{}{}
	for _, c := range out {
		d := strings.TrimSpace(strings.ToLower(c.OfficialDomain))
		if d != "" {
			seenDomain[d] = struct{}{}
		}
		n := strings.TrimSpace(strings.ToLower(companyNameFromContext(c.UnstructuredContext)))
		if n != "" {
			seenCompany[n] = struct{}{}
		}
	}

	seq := 0
	for _, e := range curatedDirectory {
		if len(out) >= limit {
			break
		}
		name := strings.TrimSpace(e.CompanyName)
		if name == "" {
			continue
		}
		nameKey := strings.ToLower(name)
		if _, ok := seenCompany[nameKey]; ok {
			continue
		}
		host := normalizeWebsiteHost(e.Website)
		if host != "" {
			if _, dup := seenDomain[strings.ToLower(host)]; dup {
				continue
			}
		}
		seq++
		ctx := strings.Join([]string{
			fmt.Sprintf("@company: %s", name),
			fmt.Sprintf("@industry: %s", strings.TrimSpace(e.Category)),
			fmt.Sprintf("@directory_source: curated_directory"),
		}, "\n")
		sourceRef := strings.TrimSpace(e.Website)
		if sourceRef == "" {
			sourceRef = "directory:" + nameKey
		}
		out = append(out, domain.RawCandidate{
			DiscoveryID:         fmt.Sprintf("dir-%d-%s", seq, slugID(name)),
			Source:              domain.SourceCompanyWebsite,
			SourceRef:           sourceRef,
			UnstructuredContext: ctx,
			OfficialDomain:      host,
			ProspectTrace: domain.ProspectTrace{
				SourceTrace: []string{"directory_discovery"},
			},
		})
		seenCompany[nameKey] = struct{}{}
		if host != "" {
			seenDomain[strings.ToLower(host)] = struct{}{}
		}
	}
	return out
}

func normalizeWebsiteHost(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return ""
	}
	return companycheck.SanitizeCompanyWebsiteDomain(u.Hostname())
}

func companyNameFromContext(ctx string) string {
	for _, line := range strings.Split(ctx, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "@company:") {
			return strings.TrimSpace(line[len("@company:"):])
		}
	}
	return ""
}

func slugID(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "&", "and")
	return s
}

