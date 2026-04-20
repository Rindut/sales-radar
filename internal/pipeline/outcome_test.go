package pipeline

import (
	"testing"

	"salesradar/internal/discovery"
	"salesradar/internal/domain"
)

func TestDecideRunOutcome_WebsiteSuccess(t *testing.T) {
	stats := RunStats{
		CandidatesFound: 10,
		RowsStored:      5,
		ProviderStatuses: []discovery.ProviderStatus{
			{ProviderName: "website_crawl_discovery", State: discovery.ProviderSuccess},
		},
	}
	out := decideRunOutcome(stats, domain.DiscoverySourceToggles{WebsiteCrawl: true})
	if out != RunOutcomeSuccess {
		t.Fatalf("expected success, got %s", out)
	}
}

func TestDecideRunOutcome_WebsiteNotConfigured(t *testing.T) {
	stats := RunStats{
		CandidatesFound: 10,
		RowsStored:      5,
		ProviderStatuses: []discovery.ProviderStatus{
			{ProviderName: "website_crawl_discovery", State: discovery.ProviderNotConfigured},
		},
	}
	out := decideRunOutcome(stats, domain.DiscoverySourceToggles{WebsiteCrawl: true})
	if out != RunOutcomePartialSuccess {
		t.Fatalf("expected partial_success, got %s", out)
	}
}

func TestDecideRunOutcome_WebsiteDegraded(t *testing.T) {
	stats := RunStats{
		CandidatesFound: 10,
		RowsStored:      5,
		ProviderStatuses: []discovery.ProviderStatus{
			{ProviderName: "website_crawl_discovery", State: discovery.ProviderDegraded, ReasonCode: "provider_timeout"},
		},
	}
	out := decideRunOutcome(stats, domain.DiscoverySourceToggles{WebsiteCrawl: true})
	if out != RunOutcomePartialSuccess {
		t.Fatalf("expected partial_success, got %s", out)
	}
}

func TestDecideRunOutcome_CoreError(t *testing.T) {
	stats := RunStats{}
	out := decideRunOutcome(stats, domain.DiscoverySourceToggles{WebsiteCrawl: true})
	if out != RunOutcomeError {
		t.Fatalf("expected error, got %s", out)
	}
}
