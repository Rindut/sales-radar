package discovery

import (
	"context"
	"testing"

	"salesradar/internal/domain"
)

func TestApplyWebsiteEnrichmentToPool_FailpointSuccess(t *testing.T) {
	t.Setenv("SALESRADAR_ENABLE_FAILPOINTS", "1")
	t.Setenv("SALESRADAR_FAILPOINT_WEBSITE_CRAWL", "success")

	toggles := domain.DiscoverySourceToggles{WebsiteCrawl: true}
	pool := []domain.RawCandidate{{DiscoveryID: "x", OfficialDomain: "example.com"}}
	_, statuses := applyWebsiteEnrichmentToPool(context.Background(), domain.RunParams{SourceToggles: &toggles}, pool, true)

	if len(statuses) != 1 {
		t.Fatalf("expected one status, got %d", len(statuses))
	}
	if statuses[0].State != ProviderSuccess {
		t.Fatalf("expected success, got %s", statuses[0].State)
	}
	if statuses[0].ReasonCode != "failpoint_website_success" {
		t.Fatalf("expected failpoint success reason, got %q", statuses[0].ReasonCode)
	}
}

func TestApplyWebsiteEnrichmentToPool_FailpointTimeout(t *testing.T) {
	t.Setenv("SALESRADAR_ENABLE_FAILPOINTS", "1")
	t.Setenv("SALESRADAR_FAILPOINT_WEBSITE_CRAWL", "timeout")

	toggles := domain.DiscoverySourceToggles{WebsiteCrawl: true}
	pool := []domain.RawCandidate{{DiscoveryID: "x", OfficialDomain: "example.com"}}
	_, statuses := applyWebsiteEnrichmentToPool(context.Background(), domain.RunParams{SourceToggles: &toggles}, pool, true)

	if len(statuses) != 1 {
		t.Fatalf("expected one status, got %d", len(statuses))
	}
	if statuses[0].State != ProviderDegraded {
		t.Fatalf("expected degraded, got %s", statuses[0].State)
	}
	if statuses[0].ReasonCode != "provider_timeout" {
		t.Fatalf("expected provider_timeout reason, got %q", statuses[0].ReasonCode)
	}
}

func TestApplyWebsiteEnrichmentToPool_FailpointError(t *testing.T) {
	t.Setenv("SALESRADAR_ENABLE_FAILPOINTS", "1")
	t.Setenv("SALESRADAR_FAILPOINT_WEBSITE_CRAWL", "error")

	toggles := domain.DiscoverySourceToggles{WebsiteCrawl: true}
	pool := []domain.RawCandidate{{DiscoveryID: "x", OfficialDomain: "example.com"}}
	_, statuses := applyWebsiteEnrichmentToPool(context.Background(), domain.RunParams{SourceToggles: &toggles}, pool, true)

	if len(statuses) != 1 {
		t.Fatalf("expected one status, got %d", len(statuses))
	}
	if statuses[0].State != ProviderFailed {
		t.Fatalf("expected failed, got %s", statuses[0].State)
	}
	if statuses[0].ReasonCode != "unknown_provider_error" {
		t.Fatalf("expected unknown_provider_error reason, got %q", statuses[0].ReasonCode)
	}
}
