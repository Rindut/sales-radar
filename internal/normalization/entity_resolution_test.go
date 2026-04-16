package normalization

import (
	"testing"

	"salesradar/internal/domain"
)

func TestDeduplicateCandidates_MergesByDomain(t *testing.T) {
	in := []domain.RawCandidate{
		{
			DiscoveryID:    "a",
			OfficialDomain: "example.com",
			UnstructuredContext: "@company: Example Corp\n" +
				"@industry: retail",
			ProspectTrace: domain.ProspectTrace{SourceTrace: []string{"seed_discovery"}},
		},
		{
			DiscoveryID:    "b",
			OfficialDomain: "example.com",
			UnstructuredContext: "@company: Example Corporation\n" +
				"@industry: retail",
			ProspectTrace: domain.ProspectTrace{SourceTrace: []string{"website_crawl_discovery"}},
		},
	}

	out := DeduplicateCandidates(in)
	if len(out) != 1 {
		t.Fatalf("expected 1 candidate after domain merge, got %d", len(out))
	}
	if len(out[0].ProspectTrace.SourceTrace) < 2 {
		t.Fatalf("expected merged source trace, got %v", out[0].ProspectTrace.SourceTrace)
	}
}

func TestDeduplicateCandidates_MergesBySimilarName(t *testing.T) {
	in := []domain.RawCandidate{
		{
			DiscoveryID: "a",
			UnstructuredContext: "@company: PT Maju Jaya Abadi\n" +
				"@industry: retail",
			ProspectTrace: domain.ProspectTrace{SourceTrace: []string{"seed_discovery"}},
		},
		{
			DiscoveryID: "b",
			UnstructuredContext: "@company: Maju Jaya Abadi PT\n" +
				"@industry: retail",
			ProspectTrace: domain.ProspectTrace{SourceTrace: []string{"job_signal_discovery"}},
		},
	}

	out := DeduplicateCandidates(in)
	if len(out) != 1 {
		t.Fatalf("expected 1 candidate after similar-name merge, got %d", len(out))
	}
}
