package normalization

import (
	"strings"

	"salesradar/internal/companycheck"
	"salesradar/internal/domain"
)

const similarNameThreshold = 0.85

// DeduplicateCandidates merges duplicate entities by domain, then by similar company name.
func DeduplicateCandidates(in []domain.RawCandidate) []domain.RawCandidate {
	if len(in) <= 1 {
		return in
	}
	out := make([]domain.RawCandidate, 0, len(in))
	byDomain := map[string]int{}
	byNameKey := map[string]int{}

	for _, c := range in {
		if idx, ok := findDuplicateIndex(c, out, byDomain, byNameKey); ok {
			out[idx] = mergeCandidates(out[idx], c)
			reindex(out[idx], idx, byDomain, byNameKey)
			continue
		}
		out = append(out, c)
		reindex(c, len(out)-1, byDomain, byNameKey)
	}
	return out
}

func findDuplicateIndex(c domain.RawCandidate, out []domain.RawCandidate, byDomain, byNameKey map[string]int) (int, bool) {
	if d := strings.TrimSpace(strings.ToLower(c.OfficialDomain)); d != "" {
		if idx, ok := byDomain[d]; ok {
			return idx, true
		}
	}
	cname := candidateCompanyName(c)
	ckey := companycheck.MergeDedupKey(cname)
	if ckey != "" {
		if idx, ok := byNameKey[ckey]; ok {
			return idx, true
		}
		for i := range out {
			oname := candidateCompanyName(out[i])
			if oname == "" {
				continue
			}
			if nameSimilarity(cname, oname) >= similarNameThreshold {
				return i, true
			}
		}
	}
	return -1, false
}

func reindex(c domain.RawCandidate, idx int, byDomain, byNameKey map[string]int) {
	if d := strings.TrimSpace(strings.ToLower(c.OfficialDomain)); d != "" {
		byDomain[d] = idx
	}
	if n := companycheck.MergeDedupKey(candidateCompanyName(c)); n != "" {
		byNameKey[n] = idx
	}
}

func mergeCandidates(a, b domain.RawCandidate) domain.RawCandidate {
	base, other := a, b
	if candidateScore(other) > candidateScore(base) {
		base, other = other, base
	}
	if strings.TrimSpace(base.OfficialDomain) == "" {
		base.OfficialDomain = strings.TrimSpace(other.OfficialDomain)
	}
	if strings.TrimSpace(base.SourceRef) == "" {
		base.SourceRef = strings.TrimSpace(other.SourceRef)
	}
	base.ProspectTrace.UsedGoogle = base.ProspectTrace.UsedGoogle || other.ProspectTrace.UsedGoogle
	base.ProspectTrace.UsedApollo = base.ProspectTrace.UsedApollo || other.ProspectTrace.UsedApollo
	base.ProspectTrace.UsedLinkedIn = base.ProspectTrace.UsedLinkedIn || other.ProspectTrace.UsedLinkedIn
	base.ProspectTrace.SourceTrace = appendUnique(base.ProspectTrace.SourceTrace, other.ProspectTrace.SourceTrace...)
	base.UnstructuredContext = mergeContext(base.UnstructuredContext, other.UnstructuredContext)
	return base
}

func candidateScore(c domain.RawCandidate) int {
	n := 0
	if strings.TrimSpace(c.OfficialDomain) != "" {
		n += 3
	}
	n += len(c.ProspectTrace.SourceTrace)
	n += len(strings.Fields(strings.TrimSpace(c.UnstructuredContext))) / 10
	return n
}

func appendUnique(base []string, add ...string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(base)+len(add))
	for _, s := range base {
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
	for _, s := range add {
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

func mergeContext(a, b string) string {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	if strings.Contains(a, b) {
		return a
	}
	if strings.Contains(b, a) {
		return b
	}
	return a + "\n" + b
}

func candidateCompanyName(c domain.RawCandidate) string {
	for _, line := range strings.Split(c.UnstructuredContext, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "@company:") {
			return strings.TrimSpace(line[len("@company:"):])
		}
	}
	return ""
}

func nameSimilarity(a, b string) float64 {
	ta := tokens(companycheck.MergeDedupKey(a))
	tb := tokens(companycheck.MergeDedupKey(b))
	if len(ta) == 0 || len(tb) == 0 {
		return 0
	}
	var inter int
	for t := range ta {
		if _, ok := tb[t]; ok {
			inter++
		}
	}
	union := len(ta) + len(tb) - inter
	if union <= 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

func tokens(s string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, t := range strings.Fields(strings.TrimSpace(s)) {
		out[t] = struct{}{}
	}
	return out
}
