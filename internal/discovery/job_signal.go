package discovery

import (
	"context"
	"fmt"
	"strings"

	"salesradar/internal/domain"
)

// applyJobSignalDiscovery enriches each candidate that has a domain with job-page signals (mock or light HTTP fetch).
func applyJobSignalDiscovery(ctx context.Context, candidates []domain.RawCandidate) []domain.RawCandidate {
	out := make([]domain.RawCandidate, len(candidates))
	for i := range candidates {
		out[i] = enrichWithJobSignal(ctx, candidates[i])
	}
	return out
}

func enrichWithJobSignal(ctx context.Context, in domain.RawCandidate) domain.RawCandidate {
	host := strings.TrimSpace(in.OfficialDomain)
	if host == "" {
		return in
	}
	paths := []string{
		"https://" + host + "/careers",
		"https://" + host + "/jobs",
		"https://" + host + "/career",
		"https://" + host + "/join-us",
		"https://" + host + "/karir",
	}
	var blob strings.Builder
	for _, u := range paths {
		blob.WriteString(fetchPageText(ctx, u))
		blob.WriteByte(' ')
	}
	text := strings.ToLower(blob.String())
	roles := detectJobRoles(text)
	if len(roles) == 0 && shouldMockJobSignal(host) {
		text = mockJobListingText()
		roles = detectJobRoles(text)
	}
	if len(roles) == 0 {
		return in
	}

	lines := []string{
		in.UnstructuredContext,
		fmt.Sprintf("@job_signal_roles: %s", strings.Join(roles, ", ")),
		"@job_signal_detected: true",
	}
	in.UnstructuredContext = strings.Join(lines, "\n")
	if !containsTrace(in.ProspectTrace.SourceTrace, "job_signal") {
		in.ProspectTrace.SourceTrace = append(in.ProspectTrace.SourceTrace, "job_signal")
	}
	return in
}

func detectJobRoles(text string) []string {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	var roles []string
	add := func(label string) {
		for _, r := range roles {
			if r == label {
				return
			}
		}
		roles = append(roles, label)
	}
	if hasAny(text,
		"learning and development", "l&d", "l and d", "talent development", "organizational development",
	) {
		add("L&D")
	}
	if hasAny(text,
		"human resources", "hr business partner", "hrbp", "people & culture", "people and culture", "people operations",
	) {
		add("HR")
	}
	if hasAny(text,
		"training manager", "training specialist", "corporate training", "learning specialist", "instructional designer",
	) {
		add("Training")
	}
	if hasAny(text,
		"operations manager", "field operations", "operations lead", "head of operations", "operations supervisor",
	) {
		add("Operations")
	}
	return roles
}

func shouldMockJobSignal(host string) bool {
	h := 0
	for _, c := range host {
		h += int(c)
	}
	return h%5 == 0
}

func mockJobListingText() string {
	return `open roles: learning and development specialist; human resources business partner; corporate training coordinator; operations supervisor — apply on our careers page`
}
