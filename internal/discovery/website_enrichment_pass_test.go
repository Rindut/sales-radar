package discovery

import (
	"context"
	"testing"

	"salesradar/internal/domain"
)

func TestApplyWebsiteEnrichmentToPool_toggleOffLeavesPoolUnchanged(t *testing.T) {
	ctx := context.Background()
	pool := []domain.RawCandidate{
		{DiscoveryID: "a", OfficialDomain: "example.com"},
	}
	toggles := domain.DiscoverySourceToggles{WebsiteCrawl: false}
	out, statuses := applyWebsiteEnrichmentToPool(ctx, domain.RunParams{SourceToggles: &toggles}, pool, true)
	if len(out) != 1 || out[0].OfficialDomain != "example.com" {
		t.Fatalf("pool mutated: %+v", out)
	}
	if len(statuses) != 1 || statuses[0].State != ProviderDisabled {
		t.Fatalf("expected disabled status, got %+v", statuses)
	}
}

func TestApplyWebsiteEnrichmentToPool_runsWhenPoolMatchesBatchLimit(t *testing.T) {
	// Regression: website enrichment must not depend on "remaining batch capacity".
	ctx := context.Background()
	toggles := domain.DiscoverySourceToggles{WebsiteCrawl: true}
	pool := []domain.RawCandidate{
		{DiscoveryID: "x", OfficialDomain: "example.invalid"},
	}
	out, statuses := applyWebsiteEnrichmentToPool(ctx, domain.RunParams{SourceToggles: &toggles}, pool, true)
	if len(out) != 1 {
		t.Fatalf("len=%d", len(out))
	}
	if len(statuses) != 1 || statuses[0].State != ProviderNotConfigured {
		t.Fatalf("expected not_configured website provider, got %+v", statuses)
	}
}
