package normalization

import (
	"strings"

	"salesradar/internal/companycheck"
	"salesradar/internal/domain"
)

// NormalizeCandidates standardizes discovery output into a consistent shape.
func NormalizeCandidates(in []domain.RawCandidate) []domain.RawCandidate {
	if len(in) == 0 {
		return nil
	}
	out := make([]domain.RawCandidate, 0, len(in))
	for _, c := range in {
		c.DiscoveryID = strings.TrimSpace(c.DiscoveryID)
		c.SourceRef = strings.TrimSpace(c.SourceRef)
		c.OfficialDomain = companycheck.SanitizeCompanyWebsiteDomain(c.OfficialDomain)
		c.UnstructuredContext = normalizeTaggedContext(c.UnstructuredContext)
		c.ProspectTrace.SourceTrace = normalizeSourceTrace(c.ProspectTrace.SourceTrace)
		if len(c.ProspectTrace.SourceTrace) == 0 {
			c.ProspectTrace.SourceTrace = []string{c.PrimaryDiscoverySourceName()}
		}
		out = append(out, c)
	}
	return out
}

func normalizeSourceTrace(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, s := range in {
		t := strings.TrimSpace(s)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}

func normalizeTaggedContext(ctx string) string {
	ctx = strings.TrimSpace(ctx)
	if ctx == "" {
		return ""
	}
	lines := strings.Split(ctx, "\n")
	for i := range lines {
		line := strings.TrimSpace(lines[i])
		switch {
		case hasTagPrefix(line, "@company:"):
			lines[i] = "@company: " + normalizeTagValue(line, "@company:")
		case hasTagPrefix(line, "@industry:"):
			lines[i] = "@industry: " + normalizeTagValue(line, "@industry:")
		default:
			lines[i] = line
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func hasTagPrefix(s, tag string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(s)), tag)
}

func normalizeTagValue(line, tag string) string {
	v := strings.TrimSpace(line[len(tag):])
	return strings.Join(strings.Fields(v), " ")
}
